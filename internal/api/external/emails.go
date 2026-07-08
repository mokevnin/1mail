package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/emailtemplate"
	"github.com/mokevnin/1mail/ent/transactionalemail"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/messaging"
	"github.com/mokevnin/1mail/internal/service"
)

// EmailsSend is the transactional send surface (ADR 0005): a single-recipient
// email rendered from a referenced Template with per-call variables at send time.
// It binds the template by reference (renders current content), respects the
// workspace Suppression list, and skips Unsubscribe — transactional mail carries
// no sending source.
//
// Every send writes a durable TransactionalEmail record (the send-history trace),
// and the optional Idempotency-Key makes retries safe: the record is the
// synchronous claim. A repeated key replays the original outcome instead of
// sending again (the Event-log DedupKey only dedupes the event row, never the
// provider call), and a key whose first request is still in flight returns 409.
func (h *Handlers) EmailsSend(ctx context.Context, req *externalapi.SendTransactionalEmailInput, params externalapi.EmailsSendParams) (externalapi.EmailsSendRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "emails:send") {
		res := externalapi.EmailsSendUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))

	dest := eligibility.NormalizeDestination(string(req.Destination))
	if dest == "" {
		res := externalapi.EmailsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "destination must not be empty"))
		return &res, nil
	}

	idemKey, hasKey := params.IdempotencyKey.Get()
	hasKey = hasKey && idemKey != ""

	// Replay: a known key short-circuits to its recorded outcome before any work.
	if hasKey {
		existing, err := h.ent.TransactionalEmail.Query().
			Where(transactionalemail.WorkspaceID(ws), transactionalemail.IdempotencyKey(idemKey)).
			Only(ctx)
		if err == nil {
			return replay(existing), nil
		}
		if !ent.IsNotFound(err) {
			return nil, err
		}
	}

	templateID, err := parseEntityID(req.TemplateId)
	if err != nil {
		res := externalapi.EmailsSendNotFound(problem(http.StatusNotFound, "template not found"))
		return &res, nil
	}
	// Workspace-scoped: another workspace's template id must 404, never send.
	tmpl, err := h.ent.EmailTemplate.Query().
		Where(emailtemplate.IDEQ(templateID), emailtemplate.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.EmailsSendNotFound(problem(http.StatusNotFound, "template not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	// The contact this destination resolves to, when one exists (transactional mail
	// may go to an address with no contact); the send fact attaches to it.
	contactID, err := service.ResolveContactID(ctx, h.ent, ws, "", &dest, nil)
	if err != nil {
		return nil, err
	}

	// Suppression is the global hard floor; transactional skips Unsubscribe. A
	// suppressed destination still records a row (status=suppressed) so the trace and
	// the idempotency key are honored — but no email and no email.sent event.
	decision, err := eligibility.CheckTransactional(ctx, h.ent, ws, eligibility.ChannelEmail, dest)
	if err != nil {
		return nil, err
	}
	if !decision.Eligible {
		rec, res, err := h.claim(ctx, ws, dest, templateID, contactID, idemKey, hasKey, transactionalemail.StatusSuppressed)
		if rec == nil {
			return res, err // conflict replay / error
		}
		return replay(rec), nil
	}

	vars, err := decodeVariables(req.Variables)
	if err != nil {
		res := externalapi.EmailsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "invalid variables: "+err.Error()))
		return &res, nil
	}

	rendered, err := emailrender.RenderEmail(tmpl.Subject, tmpl.Body, vars)
	if err != nil {
		res := externalapi.EmailsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "render template: "+err.Error()))
		return &res, nil
	}

	// From left empty: the provider falls back to the integration's configured
	// sender (same as the automation send step).
	sender, err := h.resolver.EmailSender(ctx, ws)
	if errors.Is(err, messaging.ErrNoProvider) {
		res := externalapi.EmailsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "no default email provider configured"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	// Insert-first claim: the pending row is taken before the provider call, so a
	// concurrent request with the same key loses the unique-index race and replays
	// (or gets 409) instead of sending a second email.
	rec, res, err := h.claim(ctx, ws, dest, templateID, contactID, idemKey, hasKey, transactionalemail.StatusPending)
	if rec == nil {
		return res, err
	}

	if err := sender.Send(ctx, messaging.EmailMessage{
		To:      dest,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
		Text:    rendered.Text,
	}); err != nil {
		_, _ = rec.Update().SetStatus(transactionalemail.StatusFailed).SetError(err.Error()).Save(ctx)
		// Verified-domain send gate (ADR 0010 slice 3): a From whose domain isn't a
		// verified sending domain is a client-correctable condition, not a 500.
		if errors.Is(err, messaging.ErrUnverifiedSendingDomain) {
			res := externalapi.EmailsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "sender domain is not a verified sending domain"))
			return &res, nil
		}
		return nil, fmt.Errorf("transactional send: %w", err)
	}

	// Mark sent + publish email.sent atomically (transactional outbox), so the send
	// fact reaches the Event log like broadcast/automation sends. DedupID keys it to
	// this record for persist idempotency under at-least-once redelivery.
	if err := h.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		if _, err := tx.TransactionalEmail.UpdateOneID(rec.ID).
			SetStatus(transactionalemail.StatusSent).Save(ctx); err != nil {
			return err
		}
		return pub.Publish(ctx, &events.EmailEngagement{
			Action:      events.NameEmailSent,
			WorkspaceID: ws,
			ContactID:   contactID,
			Email:       dest,
			DedupID:     fmt.Sprintf("email.sent:transactional:%d", rec.ID),
		})
	}); err != nil {
		return nil, err
	}
	rec.Status = transactionalemail.StatusSent
	return replay(rec), nil
}

// claim inserts the send record in the given terminal-or-pending state. On the
// unique (workspace, idempotency_key) race it returns (nil, replayOrConflict, nil)
// for the loser; otherwise it returns the created record. A keyless send never
// conflicts (NULL keys are distinct in Postgres).
func (h *Handlers) claim(ctx context.Context, ws int64, dest string, templateID, contactID int64, idemKey string, hasKey bool, status transactionalemail.Status) (*ent.TransactionalEmail, externalapi.EmailsSendRes, error) {
	create := h.ent.TransactionalEmail.Create().
		SetWorkspaceID(ws).
		SetChannel(transactionalemail.ChannelEmail).
		SetDestination(dest).
		SetTemplateID(templateID).
		SetStatus(status)
	if contactID != 0 {
		create.SetContactID(contactID)
	}
	if hasKey {
		create.SetIdempotencyKey(idemKey)
	}
	rec, err := create.Save(ctx)
	if err == nil {
		return rec, nil, nil
	}
	if hasKey && ent.IsConstraintError(err) {
		// Lost the race: another request claimed this key. Replay its outcome.
		existing, gerr := h.ent.TransactionalEmail.Query().
			Where(transactionalemail.WorkspaceID(ws), transactionalemail.IdempotencyKey(idemKey)).
			Only(ctx)
		if gerr != nil {
			return nil, nil, gerr
		}
		return nil, replay(existing), nil
	}
	return nil, nil, err
}

// replay maps a send record to the API result. Terminal sent/suppressed return
// 202 with the recorded outcome; a still-pending claim (a concurrent in-flight
// request) or a prior failure return 409 — a failed key is spent, so the caller
// should retry under a fresh Idempotency-Key.
func replay(rec *ent.TransactionalEmail) externalapi.EmailsSendRes {
	switch rec.Status {
	case transactionalemail.StatusSent:
		return &externalapi.SendTransactionalEmailResponse{
			ID: entityID(rec.ID), Status: externalapi.TransactionalSendStatusSent, Destination: rec.Destination,
		}
	case transactionalemail.StatusSuppressed:
		return &externalapi.SendTransactionalEmailResponse{
			ID: entityID(rec.ID), Status: externalapi.TransactionalSendStatusSuppressed, Destination: rec.Destination,
		}
	case transactionalemail.StatusFailed:
		res := externalapi.EmailsSendConflict(problem(http.StatusConflict, "a previous send with this Idempotency-Key failed; retry with a new key"))
		return &res
	default: // pending
		res := externalapi.EmailsSendConflict(problem(http.StatusConflict, "a send with this Idempotency-Key is already in progress"))
		return &res
	}
}

// entityID renders an int64 primary key as the API's string EntityId.
func entityID(id int64) externalapi.EntityId {
	return externalapi.EntityId(strconv.FormatInt(id, 10))
}

// decodeVariables turns the per-call variables (raw JSON values) into the binding
// map Liquid renders against. Each value is decoded to its natural Go type
// (string/number/bool/nested) so merge tags like {{ amount }} render correctly.
func decodeVariables(opt externalapi.OptSendTransactionalEmailInputVariables) (map[string]any, error) {
	raw, ok := opt.Get()
	if !ok || len(raw) == 0 {
		return map[string]any{}, nil
	}
	out := make(map[string]any, len(raw))
	for k, v := range raw {
		var val any
		if err := json.Unmarshal([]byte(v), &val); err != nil {
			return nil, fmt.Errorf("variable %q: %w", k, err)
		}
		out[k] = val
	}
	return out, nil
}
