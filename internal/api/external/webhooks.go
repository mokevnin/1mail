package external

import (
	"context"
	"net/http"
	"slices"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/webhookendpoint"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

// webhookResource builds the API resource by hand (not goverter): the signing
// secret is deliberately never part of an /api response, and a nil event filter
// must render as an empty list.
func webhookResource(e *ent.WebhookEndpoint) externalapi.WebhookResource {
	return externalapi.WebhookResource{
		ID:         externalapi.EntityId(strconv.FormatInt(e.ID, 10)),
		URL:        e.URL,
		EventTypes: lo.Ternary(e.EventTypes == nil, []string{}, e.EventTypes),
		Enabled:    e.Enabled,
		CreatedAt:  externalapi.Timestamp(e.CreatedAt),
		UpdatedAt:  externalapi.Timestamp(e.UpdatedAt),
	}
}

func urlProblem() externalapi.ProblemDetails {
	p := problem(http.StatusUnprocessableEntity, "url must be an absolute http or https URL")
	p.Errors = externalapi.NewOptProblemDetailsErrors(externalapi.ProblemDetailsErrors{"url": {"must be an absolute http or https URL"}})
	return p
}

// auditEntryUnlicensed reports whether types selects audit.entry on an instance
// without an Enterprise license (ADR 0022: forwarding is EE only).
func (h *Handlers) auditEntryUnlicensed(types []string) bool {
	return slices.Contains(types, events.NameAuditEntry) && (h.audit == nil || !h.audit.Licensed())
}

func auditEntryProblem() externalapi.ProblemDetails {
	const detail = "audit.entry needs an Enterprise license"
	p := problem(http.StatusUnprocessableEntity, detail)
	p.Errors = externalapi.NewOptProblemDetailsErrors(externalapi.ProblemDetailsErrors{"eventTypes": {detail}})
	return p
}

func (h *Handlers) WebhooksList(ctx context.Context, params externalapi.WebhooksListParams) (externalapi.WebhooksListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "webhooks:read") {
		res := externalapi.WebhooksListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := scoped.WebhookEndpoint().Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Asc(webhookendpoint.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	return &externalapi.WebhooksListOK{
		Items:      lo.Map(items, func(e *ent.WebhookEndpoint, _ int) externalapi.WebhookResource { return webhookResource(e) }),
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) WebhooksCreate(ctx context.Context, req *externalapi.CreateWebhookInput) (externalapi.WebhooksCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "webhooks:write") {
		res := externalapi.WebhooksCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}
	if !service.ValidWebhookURL(req.URL) {
		res := externalapi.WebhooksCreateUnprocessableEntity(urlProblem())
		return &res, nil
	}

	if h.auditEntryUnlicensed(req.EventTypes) {
		res := externalapi.WebhooksCreateUnprocessableEntity(auditEntryProblem())
		return &res, nil
	}

	// The signing secret is generated server-side and kept sealed; it is shown
	// only in the app, never through /api.
	secret, err := service.GenerateWebhookSecret()
	if err != nil {
		return nil, err
	}
	sealed, err := h.cipher.Encrypt([]byte(secret))
	if err != nil {
		return nil, err
	}

	scoped := auth.TokenScoped(ctx)
	q := scoped.WebhookEndpoint().Create().
		SetURL(req.URL).
		SetSecretEncrypted(sealed).
		SetEventTypes(req.EventTypes).
		SetNillableEnabled(convert.Ptr(req.Enabled))
	e, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	res := webhookResource(e)
	return &res, nil
}

func (h *Handlers) WebhooksGet(ctx context.Context, params externalapi.WebhooksGetParams) (externalapi.WebhooksGetRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "webhooks:read") {
		res := externalapi.WebhooksGetUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.WebhooksGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	e, err := scoped.WebhookEndpoint().Get(ctx, id)
	if ent.IsNotFound(err) {
		res := externalapi.WebhooksGetNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := webhookResource(e)
	return &res, nil
}

func (h *Handlers) WebhooksUpdate(ctx context.Context, req *externalapi.UpdateWebhookInput, params externalapi.WebhooksUpdateParams) (externalapi.WebhooksUpdateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "webhooks:write") {
		res := externalapi.WebhooksUpdateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.WebhooksUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	if v, ok := req.URL.Get(); ok && !service.ValidWebhookURL(v) {
		res := externalapi.WebhooksUpdateUnprocessableEntity(urlProblem())
		return &res, nil
	}

	if h.auditEntryUnlicensed(req.EventTypes) {
		res := externalapi.WebhooksUpdateUnprocessableEntity(auditEntryProblem())
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	upd := scoped.WebhookEndpoint().UpdateOneID(id).
		SetNillableURL(convert.StringPtr(req.URL)).
		SetNillableEnabled(convert.Ptr(req.Enabled))
	if req.EventTypes != nil {
		upd = upd.SetEventTypes(req.EventTypes)
	}
	e, err := upd.Save(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.WebhooksUpdateNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := webhookResource(e)
	return &res, nil
}

func (h *Handlers) WebhooksDelete(ctx context.Context, params externalapi.WebhooksDeleteParams) (externalapi.WebhooksDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "webhooks:write") {
		res := externalapi.WebhooksDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.WebhooksDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	err = scoped.WebhookEndpoint().DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.WebhooksDeleteNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.WebhooksDeleteNoContent{}, nil
}
