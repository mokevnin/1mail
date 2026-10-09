package outbound

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/outboundmessage"
	"github.com/mokevnin/1mail/internal/logging"
)

// find loads the Outbound message recorded under (workspace, key), or nil.
func (m *Module) find(ctx context.Context, workspaceID int64, key string) (*ent.OutboundMessage, error) {
	msg, err := m.ent.OutboundMessage.Query().
		Where(outboundmessage.WorkspaceID(workspaceID), outboundmessage.IdempotencyKey(key)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return msg, err
}

// claim takes the right to send under req.Key: it inserts the pending row before
// the provider is called. If a row already exists (existing, from Send's replay
// check, or one that appears in a race) it is replayed when final, taken over when
// its lease has expired, and otherwise reported as ErrInProgress.
func (m *Module) claim(ctx context.Context, req Request, dest string, g gateResult, existing *ent.OutboundMessage) (*ent.OutboundMessage, *Result, error) {
	if existing != nil {
		msg, err := m.takeOver(ctx, existing)
		return msg, nil, err
	}

	create := m.ent.OutboundMessage.Create().
		SetWorkspaceID(req.WorkspaceID).
		SetKind(req.Kind).
		SetIdempotencyKey(req.Key).
		SetDestination(dest).
		SetClaimedAt(time.Now())
	if g.domain != "" {
		create.SetSendingDomain(g.domain)
	}
	if req.Source != "" {
		create.SetSendingSource(req.Source)
	}
	contactID := req.ContactID
	if req.Contact != nil {
		contactID = req.Contact.ID
	}
	if contactID != 0 {
		create.SetContactID(contactID)
	}
	ref := req.Ref
	if ref.BroadcastID != 0 {
		create.SetBroadcastID(ref.BroadcastID)
	}
	if ref.BroadcastRecipient != 0 {
		create.SetBroadcastRecipientID(ref.BroadcastRecipient)
	}
	if ref.AutomationID != 0 {
		create.SetAutomationID(ref.AutomationID)
	}
	if ref.AutomationRunID != 0 {
		create.SetAutomationRunID(ref.AutomationRunID)
	}
	if ref.AutomationStep != nil {
		create.SetAutomationStep(*ref.AutomationStep)
	}
	if ref.TemplateID != 0 {
		create.SetTemplateID(ref.TemplateID)
	}

	msg, err := create.Save(ctx)
	if err == nil {
		return msg, nil, nil
	}
	if !ent.IsConstraintError(err) {
		return nil, nil, err
	}
	// Lost the insert race: another attempt claimed this key between our replay
	// check and now. Treat its row exactly like one we found up front.
	other, ferr := m.find(ctx, req.WorkspaceID, req.Key)
	if ferr != nil {
		return nil, nil, ferr
	}
	if other == nil {
		return nil, nil, err
	}
	if other.Status != outboundmessage.StatusPending {
		r := replayResult(other)
		return nil, &r, nil
	}
	msg, terr := m.takeOver(ctx, other)
	return msg, nil, terr
}

// takeOver adopts a pending claim, but only when its lease has expired (a crashed
// or abandoned attempt). The update is conditional on the claim being exactly the
// one we read, so of several racing retries exactly one wins; everyone else, and
// any retry that finds a live claim, gets ErrInProgress.
func (m *Module) takeOver(ctx context.Context, existing *ent.OutboundMessage) (*ent.OutboundMessage, error) {
	if time.Since(existing.ClaimedAt) < m.lease {
		return nil, ErrInProgress
	}
	n, err := m.ent.OutboundMessage.Update().
		Where(
			outboundmessage.ID(existing.ID),
			outboundmessage.StatusEQ(outboundmessage.StatusPending),
			outboundmessage.ClaimedAtEQ(existing.ClaimedAt),
		).
		SetClaimedAt(time.Now()).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, ErrInProgress
	}
	return m.ent.OutboundMessage.Get(ctx, existing.ID)
}

// release hands the claim back after a definite provider failure (the call
// returned), so the next attempt may retry immediately instead of waiting out the
// lease. Best effort: if it fails, the lease expires on its own.
func (m *Module) release(ctx context.Context, msg *ent.OutboundMessage) {
	if err := m.ent.OutboundMessage.UpdateOneID(msg.ID).
		SetClaimedAt(time.Unix(0, 0)).
		Exec(ctx); err != nil {
		logging.FromContext(ctx).Warn("outbound: release claim failed", "message_id", msg.ID, "err", err)
	}
}

// finish records a final, non-sent status with its reason.
func (m *Module) finish(ctx context.Context, msg *ent.OutboundMessage, status outboundmessage.Status, outcome Outcome, reason string) (Result, error) {
	if err := m.ent.OutboundMessage.UpdateOneID(msg.ID).
		SetStatus(status).
		SetReason(reason).
		Exec(ctx); err != nil {
		return Result{}, err
	}
	return Result{Outcome: outcome, Reason: reason, MessageID: msg.ID}, nil
}
