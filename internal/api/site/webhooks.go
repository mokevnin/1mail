package site

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/webhookendpoint"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

// webhookResource builds the API resource, decrypting the signing secret so the
// UI can display it for signature verification. Built by hand (not goverter)
// because the secret is stored encrypted.
func (h *Handlers) webhookResource(e *ent.WebhookEndpoint) (siteapi.SiteWebhookEndpointResource, error) {
	secret, err := h.cipher.Decrypt(e.SecretEncrypted)
	if err != nil {
		return siteapi.SiteWebhookEndpointResource{}, err
	}
	types := e.EventTypes
	if types == nil {
		types = []string{}
	}
	return siteapi.SiteWebhookEndpointResource{
		ID:         siteapi.EntityId(strconv.FormatInt(e.ID, 10)),
		URL:        e.URL,
		Secret:     string(secret),
		EventTypes: types,
		Enabled:    e.Enabled,
		CreatedAt:  siteapi.Timestamp(e.CreatedAt),
		UpdatedAt:  siteapi.Timestamp(e.UpdatedAt),
	}, nil
}

// validWebhookURL accepts only absolute http(s) URLs. (Network-level SSRF
// defenses live in the delivery worker, which dials the resolved IP.)
func validWebhookURL(raw string) bool {
	u, err := url.Parse(raw)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (h *Handlers) SiteWebhooksList(ctx context.Context, params siteapi.SiteWebhooksListParams) (siteapi.SiteWebhooksListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	var pagePtr, pageSizePtr *int32
	if v, ok := params.Page.Get(); ok {
		pagePtr = &v
	}
	if v, ok := params.PageSize.Get(); ok {
		pageSizePtr = &v
	}
	page, pageSize := pagination.Normalize(pagePtr, pageSizePtr)

	q := h.ent.WebhookEndpoint.Query().Where(webhookendpoint.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(webhookendpoint.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteWebhookEndpointResource, len(items))
	for i, e := range items {
		res, err := h.webhookResource(e)
		if err != nil {
			return nil, err
		}
		resources[i] = res
	}
	return &siteapi.SiteWebhooksListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteWebhooksCreate(ctx context.Context, req *siteapi.SiteCreateWebhookEndpointInput, params siteapi.SiteWebhooksCreateParams) (siteapi.SiteWebhooksCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	if !validWebhookURL(req.URL) {
		v := siteapi.SiteWebhooksCreateUnprocessableEntity(problemWithErrors(http.StatusUnprocessableEntity, i18n.T("errors.url_invalid", nil), map[string][]string{
			"url": {i18n.T("errors.url_must_be_absolute", nil)},
		}))
		return &v, nil
	}

	secret, err := service.GenerateWebhookSecret()
	if err != nil {
		return nil, err
	}
	encrypted, err := h.cipher.Encrypt([]byte(secret))
	if err != nil {
		return nil, err
	}

	q := h.ent.WebhookEndpoint.Create().
		SetWorkspaceID(ws).
		SetURL(req.URL).
		SetSecretEncrypted(encrypted).
		SetEventTypes(req.EventTypes)
	if v, ok := req.Enabled.Get(); ok {
		q = q.SetEnabled(v)
	}
	e, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.webhookResource(e)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (h *Handlers) SiteWebhooksGet(ctx context.Context, params siteapi.SiteWebhooksGetParams) (siteapi.SiteWebhooksGetRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteWebhooksGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	e, err := h.ent.WebhookEndpoint.Query().
		Where(webhookendpoint.IDEQ(id), webhookendpoint.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksGetNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := h.webhookResource(e)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (h *Handlers) SiteWebhooksUpdate(ctx context.Context, req *siteapi.SiteUpdateWebhookEndpointInput, params siteapi.SiteWebhooksUpdateParams) (siteapi.SiteWebhooksUpdateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteWebhooksUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	upd := h.ent.WebhookEndpoint.UpdateOneID(id).Where(webhookendpoint.WorkspaceID(ws))
	if v, ok := req.URL.Get(); ok {
		if !validWebhookURL(v) {
			r := siteapi.SiteWebhooksUpdateUnprocessableEntity(problemWithErrors(http.StatusUnprocessableEntity, i18n.T("errors.url_invalid", nil), map[string][]string{
				"url": {i18n.T("errors.url_must_be_absolute", nil)},
			}))
			return &r, nil
		}
		upd = upd.SetURL(v)
	}
	if v, ok := req.Enabled.Get(); ok {
		upd = upd.SetEnabled(v)
	}
	if req.EventTypes != nil {
		upd = upd.SetEventTypes(req.EventTypes)
	}
	e, err := upd.Save(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksUpdateNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := h.webhookResource(e)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (h *Handlers) SiteWebhooksDelete(ctx context.Context, params siteapi.SiteWebhooksDeleteParams) (siteapi.SiteWebhooksDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteWebhooksDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.WebhookEndpoint.DeleteOneID(id).Where(webhookendpoint.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWebhooksDeleteNotFound(problem(http.StatusNotFound, "webhook not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteWebhooksDeleteNoContent{}, nil
}
