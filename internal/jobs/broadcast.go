package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/segment"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/tracking"
)

// recipientInsertChunk bounds a single CreateBulk / InsertMany so a very large
// audience is planned in batches rather than one giant statement.
const recipientInsertChunk = 1000

// recipientMaxAttempts caps per-recipient send retries. A hard bounce won't
// recover, so the default 25 is wrong here; the failed recipient row records
// the last error.
const recipientMaxAttempts = 3

// SendBroadcastArgs is the fan-out (plan) job payload: which broadcast to
// dispatch. The worker resolves the audience and enqueues one per-recipient job
// each, so no single job carries the whole send.
type SendBroadcastArgs struct {
	BroadcastID int64 `json:"broadcast_id"`
}

func (SendBroadcastArgs) Kind() string { return "send_broadcast" }

// SendRecipientArgs is one per-recipient send. BroadcastID rides along so the
// worker can finalize without an extra lookup.
type SendRecipientArgs struct {
	RecipientID int64 `json:"recipient_id"`
	BroadcastID int64 `json:"broadcast_id"`
}

func (SendRecipientArgs) Kind() string { return "send_broadcast_recipient" }

// SenderResolver resolves a workspace's email sender. *messaging.Resolver
// satisfies it; tests pass a fake to exercise the engine without real SMTP.
type SenderResolver interface {
	EmailSender(ctx context.Context, workspaceID int64) (messaging.EmailSender, error)
}

// SendBroadcastWorker is the fan-out phase: it plans the audience and enqueues a
// per-recipient job for each, so the send scales across workers instead of
// running the whole audience in one job under the default 1-minute JobTimeout.
type SendBroadcastWorker struct {
	river.WorkerDefaults[SendBroadcastArgs]
	ent      *ent.Client
	bus      *events.Bus
	resolver SenderResolver
	tracker  *tracking.Tracker
}

// Timeout gives the plan phase room to resolve a large audience and enqueue the
// per-recipient jobs; the actual sending happens in those jobs, each bounded by
// the default per-job timeout.
func (w *SendBroadcastWorker) Timeout(*river.Job[SendBroadcastArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *SendBroadcastWorker) Work(ctx context.Context, job *river.Job[SendBroadcastArgs]) error {
	ids, err := PlanBroadcast(ctx, w.ent, w.resolver, job.Args.BroadcastID)
	if err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil // empty audience: PlanBroadcast already finalized the broadcast
	}
	rc := river.ClientFromContext[pgx.Tx](ctx)
	for i := 0; i < len(ids); i += recipientInsertChunk {
		end := min(i+recipientInsertChunk, len(ids))
		params := make([]river.InsertManyParams, 0, end-i)
		for _, id := range ids[i:end] {
			params = append(params, river.InsertManyParams{
				Args:       SendRecipientArgs{RecipientID: id, BroadcastID: job.Args.BroadcastID},
				InsertOpts: &river.InsertOpts{Queue: QueueBroadcasts, MaxAttempts: recipientMaxAttempts},
			})
		}
		if _, err := rc.InsertMany(ctx, params); err != nil {
			return err
		}
	}
	return nil
}

// SendRecipientWorker delivers one broadcast recipient. Each recipient is its
// own job with its own retry budget, so a slow or failing send never blocks the
// rest of the audience.
type SendRecipientWorker struct {
	river.WorkerDefaults[SendRecipientArgs]
	ent      *ent.Client
	bus      *events.Bus
	resolver SenderResolver
	tracker  *tracking.Tracker
}

func (w *SendRecipientWorker) Work(ctx context.Context, job *river.Job[SendRecipientArgs]) error {
	if err := SendToRecipient(ctx, w.ent, w.bus, w.resolver, w.tracker, job.Args.RecipientID); err != nil {
		// On the final attempt, record the terminal failure so the broadcast can
		// finalize instead of hanging in "sending", then surface the error (river
		// discards the job and the ErrorHandler logs it).
		if job.Attempt >= job.MaxAttempts {
			_ = markRecipientFailed(ctx, w.ent, job.Args.RecipientID, err)
			_ = FinalizeBroadcast(ctx, w.ent, job.Args.BroadcastID)
		}
		return err
	}
	// Cheap conditional no-op until the last recipient lands, at which point it
	// flips the broadcast to "sent" — no separate finalizer job needed.
	return FinalizeBroadcast(ctx, w.ent, job.Args.BroadcastID)
}

// SendBroadcast renders and sends a broadcast to its eligible audience
// synchronously (plan → send each → finalize). It is the pure, queue-free path:
// the river workers reuse the same PlanBroadcast/SendToRecipient/FinalizeBroadcast
// functions but fan the per-recipient sends out across jobs, while this composed
// form drives the whole send in one call for the inline adapter and tests.
//
// Audience = workspace contacts with an email address (optionally narrowed by a
// segment) minus the ineligible (suppressed / unsubscribed, derived per ADR 0001).
func SendBroadcast(ctx context.Context, client *ent.Client, bus *events.Bus, resolver SenderResolver, tracker *tracking.Tracker, broadcastID int64) error {
	ids, err := PlanBroadcast(ctx, client, resolver, broadcastID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if serr := SendToRecipient(ctx, client, bus, resolver, tracker, id); serr != nil {
			// Synchronous path has no retry runtime — a failed send is terminal, so
			// mark the row failed (a bad recipient must not abort the batch).
			_ = markRecipientFailed(ctx, client, id, serr)
		}
	}
	return FinalizeBroadcast(ctx, client, broadcastID)
}

// PlanBroadcast resolves the broadcast's eligible audience, creates one pending
// BroadcastRecipient row per recipient (idempotent via the unique
// (broadcast, contact) index, so a re-planned job converges), and returns the
// recipient row IDs to send. It fails fast — before any rows exist — if the
// workspace has no usable sender, and finalizes immediately on an empty audience
// so the broadcast never hangs in "sending".
func PlanBroadcast(ctx context.Context, client *ent.Client, resolver SenderResolver, broadcastID int64) ([]int64, error) {
	b, err := client.Broadcast.Get(ctx, broadcastID)
	if err != nil {
		return nil, fmt.Errorf("load broadcast %d: %w", broadcastID, err)
	}

	// Validate the sender up front: no usable provider means the whole broadcast
	// fails, and we'd rather reflect that before creating any recipient rows.
	if _, err := resolver.EmailSender(ctx, b.WorkspaceID); err != nil {
		_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
		return nil, fmt.Errorf("resolve sender for broadcast %d: %w", b.ID, err)
	}

	// Verified-domain send gate (ADR 0010 slice 3): when the broadcast pins an
	// explicit From, reject the whole broadcast up front if its domain isn't a
	// verified sending domain — rather than failing every recipient at send time.
	// A nil From falls back to the integration's configured sender, whose effective
	// domain is only known inside the provider; that case is left to the send-time
	// gate in BuildSignedMIME and (known residual) fails each recipient job over its
	// retry budget instead of failing fast here.
	if b.FromEmail != nil {
		ok, err := messaging.HasVerifiedSendingDomain(ctx, client, b.WorkspaceID, *b.FromEmail)
		if err != nil {
			return nil, fmt.Errorf("check sending domain for broadcast %d: %w", b.ID, err)
		}
		if !ok {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return nil, fmt.Errorf("broadcast %d: %w", b.ID, messaging.ErrUnverifiedSendingDomain)
		}
	}

	if b, err = b.Update().SetStatus(broadcast.StatusSending).Save(ctx); err != nil {
		return nil, err
	}

	// Audience: contacts in the workspace with an email address, narrowed by the
	// broadcast's segment when set, then filtered to eligible recipients.
	// Eligibility is derived (ADR 0001) — suppressed destinations and ones
	// unsubscribed from the "broadcasts" source (or from everything) are excluded
	// at the query level. EmailNotNil keeps un-sendable contacts out so no row is
	// created that could never send and would block finalization forever.
	audience := client.Contact.Query().Where(
		contact.WorkspaceID(b.WorkspaceID),
		contact.EmailNotNil(),
		eligibility.Predicate(eligibility.ChannelEmail, eligibility.SourceBroadcasts),
	)
	if b.SegmentID != nil {
		seg, err := client.Segment.Query().
			Where(segment.IDEQ(*b.SegmentID), segment.WorkspaceID(b.WorkspaceID)).
			Only(ctx)
		if err != nil {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return nil, fmt.Errorf("load segment %d: %w", *b.SegmentID, err)
		}
		if seg.Type != segment.TypeRule {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return nil, fmt.Errorf("segment %d: only rule segments are supported as broadcast audiences", seg.ID)
		}
		def := ""
		if seg.Definition != nil {
			def = *seg.Definition
		}
		pred, err := segments.ContactPredicate(def)
		if err != nil {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return nil, fmt.Errorf("segment %d definition: %w", seg.ID, err)
		}
		audience = audience.Where(pred)
	}

	contactIDs, err := audience.IDs(ctx)
	if err != nil {
		return nil, err
	}

	// Create one pending row per recipient, batched. OnConflict-ignore makes a
	// re-planned (retried) job converge instead of tripping the unique index.
	for i := 0; i < len(contactIDs); i += recipientInsertChunk {
		end := min(i+recipientInsertChunk, len(contactIDs))
		builders := make([]*ent.BroadcastRecipientCreate, 0, end-i)
		for _, cid := range contactIDs[i:end] {
			builders = append(builders, client.BroadcastRecipient.Create().
				SetBroadcastID(b.ID).
				SetWorkspaceID(b.WorkspaceID).
				SetContactID(cid))
		}
		if err := client.BroadcastRecipient.CreateBulk(builders...).
			OnConflictColumns("broadcast_id", "contact_id").
			Ignore().
			Exec(ctx); err != nil {
			return nil, err
		}
	}

	// Re-read the full row set so the returned IDs are complete and retry-safe
	// (independent of which rows this attempt actually inserted).
	ids, err := client.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(b.ID)).
		IDs(ctx)
	if err != nil {
		return nil, err
	}

	if _, err := b.Update().SetRecipientsTotal(len(ids)).Save(ctx); err != nil {
		return nil, err
	}

	if len(ids) == 0 {
		if err := FinalizeBroadcast(ctx, client, b.ID); err != nil {
			return nil, err
		}
	}
	return ids, nil
}

// SendToRecipient renders and sends one broadcast recipient's email, recording
// the outcome. It is idempotent: an already-sent recipient is skipped (so a
// retry after a successful send never re-sends), and success records delivery +
// publishes email.sent atomically via the transactional outbox. A send error is
// returned (not marked failed) so the caller can retry; a render error is
// terminal.
func SendToRecipient(ctx context.Context, client *ent.Client, bus *events.Bus, resolver SenderResolver, tracker *tracking.Tracker, recipientID int64) error {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("load recipient %d: %w", recipientID, err)
	}
	if rec.Status == broadcastrecipient.StatusSent {
		return nil // already delivered (retry after a committed send) — don't re-send
	}

	b, err := client.Broadcast.Get(ctx, rec.BroadcastID)
	if err != nil {
		return fmt.Errorf("load broadcast %d: %w", rec.BroadcastID, err)
	}
	c, err := client.Contact.Get(ctx, rec.ContactID)
	if err != nil {
		return fmt.Errorf("load contact %d: %w", rec.ContactID, err)
	}
	// Guard: the audience filters EmailNotNil, but a contact could lose its email
	// between plan and send. Terminal — the row can't ever send.
	if c.Email == nil {
		_, _ = rec.Update().SetStatus(broadcastrecipient.StatusFailed).SetError("contact has no email").Save(ctx)
		return nil
	}
	addr := *c.Email

	sender, err := resolver.EmailSender(ctx, rec.WorkspaceID)
	if err != nil {
		return fmt.Errorf("resolve sender: %w", err) // retryable — provider config may recover
	}

	email, rerr := emailrender.RenderEmail(b.Subject, b.Body, contactBindings(c))
	if rerr != nil {
		_, _ = rec.Update().SetStatus(broadcastrecipient.StatusFailed).SetError(rerr.Error()).Save(ctx)
		return nil // render errors are deterministic — retrying won't help
	}

	// Pipeline: liquid merge tags → (mjml compile) → CSS inline. Tracking is
	// layered on AFTER, so re-parsing can't mangle the pixel/links.
	html := email.HTML
	if tracker != nil {
		unsub := tracking.UnsubTarget{
			Source:      eligibility.SourceBroadcasts,
			Destination: addr,
			WorkspaceID: b.WorkspaceID,
			ContactID:   c.ID,
			BroadcastID: b.ID,
		}
		if tracked, terr := tracker.Rewrite(html, rec.ID, unsub); terr != nil {
			logging.FromContext(ctx).Error("broadcast: rewrite links failed", "broadcast_id", b.ID, "recipient_id", rec.ID, "err", terr)
		} else {
			html = tracked
		}
	}

	var fromEmail, fromName string
	if b.FromEmail != nil {
		fromEmail = *b.FromEmail
	}
	if b.FromName != nil {
		fromName = *b.FromName
	}

	if serr := sender.Send(ctx, messaging.EmailMessage{
		From:     fromEmail,
		FromName: fromName,
		To:       addr,
		Subject:  email.Subject,
		HTML:     html,
		Text:     email.Text,
	}); serr != nil {
		return fmt.Errorf("send to %s: %w", addr, serr) // retryable
	}

	// Record delivery + publish email.sent atomically (transactional outbox), so
	// the send fact lands in the Event log iff the recipient is marked sent.
	// The recipient row's unique (broadcast, contact) index plus the sent-skip
	// above give exactly-once; the deterministic DedupID is defense in depth.
	return bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		if _, err := tx.BroadcastRecipient.UpdateOneID(rec.ID).
			SetStatus(broadcastrecipient.StatusSent).
			SetSentAt(time.Now()).Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action:      events.NameEmailSent,
			WorkspaceID: b.WorkspaceID,
			ContactID:   c.ID,
			Email:       addr,
			BroadcastID: b.ID,
			DedupID:     fmt.Sprintf("email.sent:%d", rec.ID),
		})
	})
}

// markRecipientFailed records a terminal delivery failure on a recipient row.
func markRecipientFailed(ctx context.Context, client *ent.Client, recipientID int64, cause error) error {
	_, err := client.BroadcastRecipient.UpdateOneID(recipientID).
		SetStatus(broadcastrecipient.StatusFailed).
		SetError(cause.Error()).
		Save(ctx)
	return err
}

// FinalizeBroadcast flips a broadcast to "sent" once none of its recipients are
// still pending, deriving the aggregate counters from the recipient rows. The
// conditional WHERE status=sending makes concurrent finalizers (one per
// per-recipient job) a no-op after the first, so sent_at is set exactly once and
// the counters self-heal against any retry drift.
func FinalizeBroadcast(ctx context.Context, client *ent.Client, broadcastID int64) error {
	pending, err := client.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(broadcastID),
			broadcastrecipient.StatusEQ(broadcastrecipient.StatusPending)).
		Count(ctx)
	if err != nil {
		return err
	}
	if pending > 0 {
		return nil // not all recipients resolved yet
	}

	sent, err := client.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(broadcastID),
			broadcastrecipient.StatusEQ(broadcastrecipient.StatusSent)).
		Count(ctx)
	if err != nil {
		return err
	}
	failed, err := client.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(broadcastID),
			broadcastrecipient.StatusEQ(broadcastrecipient.StatusFailed)).
		Count(ctx)
	if err != nil {
		return err
	}

	_, err = client.Broadcast.Update().
		Where(broadcast.IDEQ(broadcastID), broadcast.StatusEQ(broadcast.StatusSending)).
		SetStatus(broadcast.StatusSent).
		SetSentAt(time.Now()).
		SetSentCount(sent).
		SetFailedCount(failed).
		Save(ctx)
	return err
}

// contactBindings builds the Liquid merge-tag context for a contact: its core
// fields plus any custom fields (custom fields can shadow nothing important and
// keep merge tags simple, e.g. {{ first_name }}, {{ company }}).
func contactBindings(c *ent.Contact) map[string]any {
	b := map[string]any{
		"email":      c.Email,
		"first_name": "",
		"last_name":  "",
	}
	if c.FirstName != nil {
		b["first_name"] = *c.FirstName
	}
	if c.LastName != nil {
		b["last_name"] = *c.LastName
	}
	for k, v := range c.CustomFields {
		b[k] = v
	}
	return b
}
