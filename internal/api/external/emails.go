package external

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/emailtemplate"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/eligibility"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/messaging"
)

// EmailsSend is the transactional send surface (ADR 0005): a single-recipient
// email rendered from a referenced Template with per-call variables at send time.
// It binds the template by reference (renders current content), respects the
// workspace Suppression list, and skips Unsubscribe — transactional mail carries
// no sending source. No campaign, no per-message record.
func (h *Handlers) EmailsSend(ctx context.Context, req *externalapi.SendTransactionalEmailInput) (externalapi.EmailsSendRes, error) {
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

	// Suppression is the global hard floor; transactional skips Unsubscribe.
	decision, err := eligibility.CheckTransactional(ctx, h.ent, ws, eligibility.ChannelEmail, dest)
	if err != nil {
		return nil, err
	}
	if !decision.Eligible {
		return &externalapi.SendTransactionalEmailResponse{
			Status:      externalapi.TransactionalSendStatusSuppressed,
			Destination: dest,
		}, nil
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
	if err := sender.Send(ctx, messaging.EmailMessage{
		To:      dest,
		Subject: rendered.Subject,
		HTML:    rendered.HTML,
		Text:    rendered.Text,
	}); err != nil {
		return nil, fmt.Errorf("transactional send: %w", err)
	}

	return &externalapi.SendTransactionalEmailResponse{
		Status:      externalapi.TransactionalSendStatusSent,
		Destination: dest,
	}, nil
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
