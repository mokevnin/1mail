package jobs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/riverqueue/river"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/segment"
	"github.com/mokevnin/1mail/ent/suppression"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/segments"
	"github.com/mokevnin/1mail/internal/tracking"
)

// SendBroadcastArgs is the river job payload: which broadcast to dispatch.
type SendBroadcastArgs struct {
	BroadcastID int64 `json:"broadcast_id"`
}

func (SendBroadcastArgs) Kind() string { return "send_broadcast" }

// SenderResolver resolves a workspace's email sender. *messaging.Resolver
// satisfies it; tests pass a fake to exercise the engine without real SMTP.
type SenderResolver interface {
	EmailSender(ctx context.Context, workspaceID int64) (messaging.EmailSender, error)
}

// SendBroadcastWorker dispatches a broadcast to its audience.
type SendBroadcastWorker struct {
	river.WorkerDefaults[SendBroadcastArgs]
	ent      *ent.Client
	resolver SenderResolver
	tracker  *tracking.Tracker
}

func (w *SendBroadcastWorker) Work(ctx context.Context, job *river.Job[SendBroadcastArgs]) error {
	return SendBroadcast(ctx, w.ent, w.resolver, w.tracker, job.Args.BroadcastID)
}

// SendBroadcast renders and sends a broadcast to all active contacts in its
// workspace, recording a BroadcastRecipient per contact and updating the
// broadcast's aggregate counters. It is exported (not just the worker) so it can
// be exercised directly in tests with a fake sender and no river runtime.
//
// Audience is all active contacts; segment targeting is a later phase. Sending
// is done inline (no per-recipient fan-out) — that's the scale path, not needed
// for the MVP.
func SendBroadcast(ctx context.Context, client *ent.Client, resolver SenderResolver, tracker *tracking.Tracker, broadcastID int64) error {
	b, err := client.Broadcast.Get(ctx, broadcastID)
	if err != nil {
		return fmt.Errorf("load broadcast %d: %w", broadcastID, err)
	}

	sender, err := resolver.EmailSender(ctx, b.WorkspaceID)
	if err != nil {
		// No usable provider: mark failed so the UI reflects it.
		_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
		return fmt.Errorf("resolve sender for broadcast %d: %w", b.ID, err)
	}

	if b, err = b.Update().SetStatus(broadcast.StatusSending).Save(ctx); err != nil {
		return err
	}

	// Audience: active contacts in the workspace, narrowed by the broadcast's
	// segment when set. Unsubscribed contacts are always excluded regardless of
	// the segment rule (compliance).
	audience := client.Contact.Query().Where(
		contact.WorkspaceID(b.WorkspaceID),
		contact.StatusEQ(contact.StatusActive),
	)
	if b.SegmentID != nil {
		seg, err := client.Segment.Query().
			Where(segment.IDEQ(*b.SegmentID), segment.WorkspaceID(b.WorkspaceID)).
			Only(ctx)
		if err != nil {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return fmt.Errorf("load segment %d: %w", *b.SegmentID, err)
		}
		if seg.Type != segment.TypeRule {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return fmt.Errorf("segment %d: only rule segments are supported as broadcast audiences", seg.ID)
		}
		def := ""
		if seg.Definition != nil {
			def = *seg.Definition
		}
		pred, err := segments.ContactPredicate(def)
		if err != nil {
			_, _ = b.Update().SetStatus(broadcast.StatusFailed).Save(ctx)
			return fmt.Errorf("segment %d definition: %w", seg.ID, err)
		}
		audience = audience.Where(pred)
	}

	contacts, err := audience.All(ctx)
	if err != nil {
		return err
	}

	// Deliverability: the suppression list is the central do-not-send registry
	// (unsubscribes, bounces, complaints, manual). Loaded once as a set and
	// checked per recipient — suppressed addresses are skipped without a
	// recipient row, independent of contact status.
	suppressed, err := suppressedEmails(ctx, client, b.WorkspaceID)
	if err != nil {
		return err
	}

	var fromEmail, fromName string
	if b.FromEmail != nil {
		fromEmail = *b.FromEmail
	}
	if b.FromName != nil {
		fromName = *b.FromName
	}

	var targeted, sentCount, failedCount int
	for _, c := range contacts {
		if _, ok := suppressed[strings.ToLower(strings.TrimSpace(c.Email))]; ok {
			continue
		}
		// Recipients targeted = audience minus suppressed. Counted independently of
		// the send outcome so a retry (where rows already exist) doesn't collapse it.
		targeted++

		// Idempotency: one recipient row per (broadcast, contact). On a retry the
		// unique index rejects the duplicate and we skip the contact.
		rec, err := client.BroadcastRecipient.Create().
			SetBroadcastID(b.ID).
			SetWorkspaceID(b.WorkspaceID).
			SetContactID(c.ID).
			Save(ctx)
		if err != nil {
			continue
		}

		// Pipeline: liquid merge tags → (mjml compile) → CSS inline. Tracking is
		// layered on AFTER, so re-parsing can't mangle the pixel/links.
		email, rerr := emailrender.RenderEmail(b.Subject, b.Body, contactBindings(c))
		if rerr != nil {
			failedCount++
			_, _ = rec.Update().
				SetStatus(broadcastrecipient.StatusFailed).
				SetError(rerr.Error()).
				Save(ctx)
			continue
		}
		html := email.HTML
		if tracker != nil {
			if tracked, terr := tracker.Rewrite(html, rec.ID); terr != nil {
				log.Printf("broadcast %d: rewrite links for recipient %d: %v", b.ID, rec.ID, terr)
			} else {
				html = tracked
			}
		}

		sendErr := sender.Send(ctx, messaging.EmailMessage{
			From:     fromEmail,
			FromName: fromName,
			To:       c.Email,
			Subject:  email.Subject,
			HTML:     html,
			Text:     email.Text,
		})
		if sendErr != nil {
			failedCount++
			_, _ = rec.Update().
				SetStatus(broadcastrecipient.StatusFailed).
				SetError(sendErr.Error()).
				Save(ctx)
			continue
		}
		sentCount++
		_, _ = rec.Update().
			SetStatus(broadcastrecipient.StatusSent).
			SetSentAt(time.Now()).
			Save(ctx)
	}

	_, err = b.Update().
		SetStatus(broadcast.StatusSent).
		SetSentAt(time.Now()).
		SetRecipientsTotal(targeted).
		SetSentCount(sentCount).
		SetFailedCount(failedCount).
		Save(ctx)
	return err
}

// suppressedEmails returns the workspace's suppression list as a set of
// normalized addresses, for an in-memory skip check during the send loop.
func suppressedEmails(ctx context.Context, client *ent.Client, workspaceID int64) (map[string]struct{}, error) {
	emails, err := client.Suppression.Query().
		Where(suppression.WorkspaceID(workspaceID)).
		Select(suppression.FieldEmail).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(emails))
	for _, e := range emails {
		// Stored emails are normalized on write; re-normalize defensively so a
		// future writer that forgets can't let a suppressed address slip through.
		set[strings.ToLower(strings.TrimSpace(e))] = struct{}{}
	}
	return set, nil
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
