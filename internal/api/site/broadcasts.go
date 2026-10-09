package site

import (
	"context"
	"errors"
	"github.com/samber/lo"
	"net/http"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	"github.com/mokevnin/1mail/ent/broadcastrecipient"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/broadcasts"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/outbound"
	"github.com/mokevnin/1mail/internal/pagination"
)

// optEntityID converts an OptNil EntityId option (a numeric string) into the
// *int64 ent's nillable setters expect. A missing or null option yields nil; a
// malformed value yields ok=false so the caller can return 400.
func optEntityID[O interface {
	Get() (siteapi.EntityId, bool)
}](o O) (id *int64, ok bool) {
	s := convert.StringPtr(o)
	if s == nil {
		return nil, true
	}
	v, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil, false
	}
	return &v, true
}

func (h *Handlers) SiteBroadcastsList(ctx context.Context, params siteapi.SiteBroadcastsListParams) (siteapi.SiteBroadcastsListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := h.ent.Broadcast.Query().Where(broadcast.WorkspaceID(ws))

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	items, err := q.Order(ent.Desc(broadcast.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteBroadcastResource, len(items))
	for i, b := range items {
		resources[i] = mapper.BroadcastToResource(b)
	}

	return &siteapi.SiteBroadcastsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteBroadcastsCreate(ctx context.Context, req *siteapi.SiteCreateBroadcastInput, params siteapi.SiteBroadcastsCreateParams) (siteapi.SiteBroadcastsCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	segmentID, ok := optEntityID(req.SegmentId)
	if !ok {
		v := siteapi.SiteBroadcastsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.segment_invalid", nil)))
		return &v, nil
	}
	integrationID, ok := optEntityID(req.IntegrationId)
	if !ok {
		v := siteapi.SiteBroadcastsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.integration_invalid", nil)))
		return &v, nil
	}

	q := h.ent.Broadcast.Create().
		SetWorkspaceID(ws).
		SetName(req.Name).
		SetNillableFromName(convert.StringPtr(req.FromName)).
		SetNillableFromEmail(convert.StringPtr(req.FromEmail)).
		SetNillableSegmentID(segmentID).
		SetNillableIntegrationID(integrationID)
	if v, ok := req.Subject.Get(); ok {
		q = q.SetSubject(v)
	}
	if v, ok := req.Body.Get(); ok {
		q = q.SetBody(v)
	}
	b, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

func (h *Handlers) SiteBroadcastsGet(ctx context.Context, params siteapi.SiteBroadcastsGetParams) (siteapi.SiteBroadcastsGetRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	b, err := h.ent.Broadcast.Query().
		Where(broadcast.IDEQ(id), broadcast.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsGetNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

func (h *Handlers) SiteBroadcastsUpdate(ctx context.Context, req *siteapi.SiteUpdateBroadcastInput, params siteapi.SiteBroadcastsUpdateParams) (siteapi.SiteBroadcastsUpdateRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	// A broadcast can only be edited while it is still a draft.
	current, err := h.ent.Broadcast.Query().
		Where(broadcast.IDEQ(id), broadcast.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsUpdateNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if current.Status != broadcast.StatusDraft {
		v := siteapi.SiteBroadcastsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "only draft broadcasts can be edited"))
		return &v, nil
	}

	segmentID, ok := optEntityID(req.SegmentId)
	if !ok {
		v := siteapi.SiteBroadcastsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.segment_invalid", nil)))
		return &v, nil
	}
	integrationID, ok := optEntityID(req.IntegrationId)
	if !ok {
		v := siteapi.SiteBroadcastsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.integration_invalid", nil)))
		return &v, nil
	}

	q := h.ent.Broadcast.UpdateOneID(id).
		Where(broadcast.WorkspaceID(ws)).
		SetNillableName(convert.StringPtr(req.Name)).
		SetNillableSubject(convert.StringPtr(req.Subject)).
		SetNillableFromName(convert.StringPtr(req.FromName)).
		SetNillableFromEmail(convert.StringPtr(req.FromEmail)).
		SetNillableBody(convert.StringPtr(req.Body)).
		SetNillableSegmentID(segmentID).
		SetNillableIntegrationID(integrationID)
	b, err := q.Save(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsUpdateNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

func (h *Handlers) SiteBroadcastsDelete(ctx context.Context, params siteapi.SiteBroadcastsDeleteParams) (siteapi.SiteBroadcastsDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	// Remove the per-recipient delivery rows first: they FK the broadcast, so a
	// sent broadcast can't be deleted while they exist. (The engagement Event log
	// keys on subject_id, not the broadcast, so it is unaffected.)
	if _, err := h.ent.BroadcastRecipient.Delete().
		Where(broadcastrecipient.BroadcastID(id), broadcastrecipient.WorkspaceID(ws)).
		Exec(ctx); err != nil {
		return nil, err
	}
	err = h.ent.Broadcast.DeleteOneID(id).Where(broadcast.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsDeleteNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteBroadcastsDeleteNoContent{}, nil
}

// SiteBroadcastsSend sends a draft or scheduled broadcast immediately. The state
// machine lives in the broadcasts module; this is the HTTP adapter.
func (h *Handlers) SiteBroadcastsSend(ctx context.Context, params siteapi.SiteBroadcastsSendParams) (siteapi.SiteBroadcastsSendRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsSendNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsSendBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	b, err := h.broadcasts.Send(ctx, ws, id)
	switch {
	case errors.Is(err, broadcasts.ErrNotFound):
		v := siteapi.SiteBroadcastsSendNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	case errors.Is(err, broadcasts.ErrNotSendable):
		v := siteapi.SiteBroadcastsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.broadcast_already_sending", nil)))
		return &v, nil
	case err != nil:
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

// SiteBroadcastsSchedule schedules a draft broadcast to send at a future time.
func (h *Handlers) SiteBroadcastsSchedule(ctx context.Context, req *siteapi.SiteScheduleBroadcastInput, params siteapi.SiteBroadcastsScheduleParams) (siteapi.SiteBroadcastsScheduleRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsScheduleNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsScheduleBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	b, err := h.broadcasts.Schedule(ctx, ws, id, time.Time(req.ScheduledAt))
	switch {
	case errors.Is(err, broadcasts.ErrNotFound):
		v := siteapi.SiteBroadcastsScheduleNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	case errors.Is(err, broadcasts.ErrNotSendable):
		v := siteapi.SiteBroadcastsScheduleUnprocessableEntity(problem(http.StatusUnprocessableEntity, i18n.T("errors.broadcast_already_sending", nil)))
		return &v, nil
	case err != nil:
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

// SiteBroadcastsTestSend renders the broadcast with sample merge data and sends
// it to a single address — no recipient rows, no tracking (a token would point
// at a nonexistent recipient). Used to preview the rendered email.
func (h *Handlers) SiteBroadcastsTestSend(ctx context.Context, req *siteapi.SiteTestSendBroadcastInput, params siteapi.SiteBroadcastsTestSendParams) (siteapi.SiteBroadcastsTestSendRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsTestSendNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteBroadcastsTestSendBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	b, err := h.ent.Broadcast.Query().
		Where(broadcast.IDEQ(id), broadcast.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsTestSendNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	to := string(req.Email)
	res, err := h.outbound.SendTest(ctx, outbound.TestRequest{
		WorkspaceID: ws,
		To:          to,
		Subject:     "[Test] " + b.Subject,
		Body:        b.Body,
		Variables:   map[string]any{"first_name": "Alex", "last_name": "Sample", "email": to},
		FromEmail:   lo.FromPtr(b.FromEmail),
		FromName:    lo.FromPtr(b.FromName),
	})
	if err != nil {
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "send failed: "+err.Error()))
		return &v, nil
	}
	switch res.Outcome {
	case outbound.Sent:
		// fall through to the 204 below
	case outbound.Held:
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, outbound.HoldDetail(res.Reason)))
		return &v, nil
	default: // outbound.Failed: the content did not render
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, res.Reason))
		return &v, nil
	}
	return &siteapi.SiteBroadcastsTestSendNoContent{}, nil
}
