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
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/ent/segment"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/segments"
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
	// ScheduledAt is the time a delayed job was enqueued for; nil for an immediate send.
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
}

func (SendBroadcastArgs) Kind() string { return "send_broadcast" }

// SendRecipientArgs is one per-recipient send. BroadcastID rides along so the
// worker can finalize without an extra lookup.
type SendRecipientArgs struct {
	RecipientID int64 `json:"recipient_id"`
	BroadcastID int64 `json:"broadcast_id"`
}

func (SendRecipientArgs) Kind() string { return "send_broadcast_recipient" }

// SendBroadcastWorker is the fan-out phase: it plans the audience and enqueues a
// per-recipient job for each, so the send scales across workers instead of
// running the whole audience in one job under the default 1-minute JobTimeout.
type SendBroadcastWorker struct {
	river.WorkerDefaults[SendBroadcastArgs]
	ent *ent.Client
	mod *outbound.Module
}

// Timeout gives the plan phase room to resolve a large audience and enqueue the
// per-recipient jobs; the actual sending happens in those jobs, each bounded by
// the default per-job timeout.
func (w *SendBroadcastWorker) Timeout(*river.Job[SendBroadcastArgs]) time.Duration {
	return 10 * time.Minute
}

func (w *SendBroadcastWorker) Work(ctx context.Context, job *river.Job[SendBroadcastArgs]) error {
	if due, err := BroadcastDue(ctx, w.ent, job.Args.BroadcastID, job.Args.ScheduledAt); err != nil || !due {
		return err // not due: unscheduled (back to draft) after this job was queued
	}
	ids, err := PlanBroadcast(ctx, w.ent, w.mod, job.Args.BroadcastID)
	if err != nil {
		return snoozeIfDeferrable(err)
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

// BroadcastDue reports whether a queued send job should still run. A Broadcast that is
// back in draft was unscheduled (or its enqueue was reverted) after the job was queued.
// A delayed job (jobScheduledAt set) is also bound to the schedule it was enqueued for:
// unscheduling leaves it queued, so after a reschedule the Broadcast carries a different
// scheduled_at (or none, once sent now) and the superseded job must not send it early.
func BroadcastDue(ctx context.Context, client *ent.Client, broadcastID int64, jobScheduledAt *time.Time) (bool, error) {
	b, err := client.Broadcast.Get(ctx, broadcastID)
	if err != nil {
		return false, fmt.Errorf("load broadcast %d: %w", broadcastID, err)
	}
	if b.Status == broadcast.StatusDraft {
		return false, nil
	}
	if jobScheduledAt == nil {
		return true, nil
	}
	// Postgres keeps microseconds; the job payload keeps nanoseconds.
	return b.ScheduledAt != nil && b.ScheduledAt.Equal(jobScheduledAt.Truncate(time.Microsecond)), nil
}

// SendRecipientWorker delivers one broadcast recipient. Each recipient is its
// own job with its own retry budget, so a slow or failing send never blocks the
// rest of the audience.
type SendRecipientWorker struct {
	river.WorkerDefaults[SendRecipientArgs]
	ent *ent.Client
	mod *outbound.Module
}

func (w *SendRecipientWorker) Work(ctx context.Context, job *river.Job[SendRecipientArgs]) error {
	if err := SendToRecipient(ctx, w.ent, w.mod, job.Args.RecipientID); err != nil {
		if _, held := asHeld(err); held {
			// A hold is not a failure: wait it out without spending an attempt.
			return snoozeIfDeferrable(err)
		}
		// On the final attempt, record the terminal failure so the broadcast can
		// finalize instead of hanging in "sending", then surface the error (river
		// discards the job and the ErrorHandler logs it).
		if job.Attempt >= job.MaxAttempts {
			_ = markRecipientFailed(ctx, w.ent, w.mod, job.Args.RecipientID, err)
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
func SendBroadcast(ctx context.Context, client *ent.Client, mod *outbound.Module, broadcastID int64) error {
	ids, err := PlanBroadcast(ctx, client, mod, broadcastID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if serr := SendToRecipient(ctx, client, mod, id); serr != nil {
			if isDeferrable(serr) {
				// The source is on hold (or a claim is live): stop, leaving the remaining
				// recipients pending for a later run rather than consuming them.
				return serr
			}
			// Synchronous path has no retry runtime — a failed send is terminal, so
			// mark the row failed (a bad recipient must not abort the batch).
			_ = markRecipientFailed(ctx, client, mod, id, serr)
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
func PlanBroadcast(ctx context.Context, client *ent.Client, mod *outbound.Module, broadcastID int64) ([]int64, error) {
	b, err := client.Broadcast.Get(ctx, broadcastID)
	if err != nil {
		return nil, fmt.Errorf("load broadcast %d: %w", broadcastID, err)
	}

	// Ask Outbound send whether this workspace can send now, before any recipient row
	// exists: a Workspace freeze, no Integration, or an unverified From domain is a
	// reversible hold (ADR 0015) — the broadcast keeps its status, records why, and
	// the plan is retried later; it is not failed.
	var from string
	if b.FromEmail != nil {
		from = *b.FromEmail
	}
	hold, err := mod.Preflight(ctx, b.WorkspaceID, from)
	if err != nil {
		return nil, fmt.Errorf("preflight broadcast %d: %w", b.ID, err)
	}
	if hold != "" {
		setBroadcastHold(ctx, client, b.ID, hold)
		return nil, &HeldError{Reason: hold}
	}
	setBroadcastHold(ctx, client, b.ID, "")

	// A template that cannot render would fail every recipient: fail the broadcast once,
	// here, before any recipient row exists (ADR 0015).
	if verr := emailrender.Validate(b.Subject, b.Body); verr != nil {
		_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
		return nil, fmt.Errorf("broadcast %d: template does not render: %w", b.ID, verr)
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

// SendToRecipient hands one broadcast recipient's email to Outbound send and maps
// the Outcome onto the recipient row. It is idempotent: a recipient already at a
// final status is left alone, and the module's idempotency key makes a retry after
// a committed send replay the recorded result. A returned error is retryable
// (provider down, claim in flight); a *HeldError means the source is on hold and
// the job should be deferred, not failed.
func SendToRecipient(ctx context.Context, client *ent.Client, mod *outbound.Module, recipientID int64) error {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return fmt.Errorf("load recipient %d: %w", recipientID, err)
	}
	if rec.Status != broadcastrecipient.StatusPending {
		return nil // already decided (retry after a committed send) — don't re-send
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

	req := outbound.Request{
		WorkspaceID: rec.WorkspaceID,
		Kind:        outboundmessage.KindBroadcast,
		Key:         recipientKey(rec.ID),
		Destination: *c.Email,
		Contact:     c,
		Source:      eligibility.SourceBroadcasts,
		Subject:     b.Subject,
		Body:        b.Body,
		TrackID:     rec.ID,
		Ref:         outbound.Ref{BroadcastID: b.ID, BroadcastRecipient: rec.ID},
	}
	if b.FromEmail != nil {
		req.FromEmail = *b.FromEmail
	}
	if b.FromName != nil {
		req.FromName = *b.FromName
	}

	res, err := mod.Send(ctx, req)
	if err != nil {
		return err
	}
	upd := rec.Update()
	if res.MessageID != 0 {
		upd.SetOutboundMessageID(res.MessageID)
	}
	switch res.Outcome {
	case outbound.Sent:
		setBroadcastHold(ctx, client, b.ID, "") // a send went through: the hold, if any, has lifted
		_, err = upd.SetStatus(broadcastrecipient.StatusSent).SetSentAt(time.Now()).Save(ctx)
	case outbound.Skipped:
		_, err = upd.SetStatus(broadcastrecipient.StatusSkipped).SetError(res.Reason).Save(ctx)
	case outbound.Failed:
		_, err = upd.SetStatus(broadcastrecipient.StatusFailed).SetError(res.Reason).Save(ctx)
	default: // outbound.Held
		setBroadcastHold(ctx, client, b.ID, res.Reason)
		return &HeldError{Reason: res.Reason}
	}
	return err
}

// recipientKey is the Outbound send idempotency key of one broadcast recipient.
func recipientKey(recipientID int64) string {
	return fmt.Sprintf("broadcast:%d", recipientID)
}

// markRecipientFailed records a terminal delivery failure on a recipient row and
// gives up the Outbound claim so a stale pending message does not linger.
func markRecipientFailed(ctx context.Context, client *ent.Client, mod *outbound.Module, recipientID int64, cause error) error {
	rec, err := client.BroadcastRecipient.Get(ctx, recipientID)
	if err != nil {
		return err
	}
	_ = mod.MarkFailed(ctx, rec.WorkspaceID, recipientKey(recipientID), cause)
	_, err = rec.Update().
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
	skipped, err := client.BroadcastRecipient.Query().
		Where(broadcastrecipient.BroadcastID(broadcastID),
			broadcastrecipient.StatusEQ(broadcastrecipient.StatusSkipped)).
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
		SetSkippedCount(skipped).
		ClearHoldReason().
		Save(ctx)
	return err
}
