package site

import (
	"context"
	"errors"
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

// broadcastResource maps a Broadcast and, while it is sending, attaches its progress
// and ETA (ADR 0023), derived on the server from its recipients.
func broadcastResource(ctx context.Context, s *ent.Scoped, b *ent.Broadcast) (siteapi.SiteBroadcastResource, error) {
	res := mapper.BroadcastToResource(b)
	p, err := broadcasts.ProgressOf(ctx, s, b, time.Now())
	if err != nil || p == nil {
		return res, err
	}
	progress := siteapi.SiteBroadcastProgress{
		ProcessedCount: int32(p.Processed),
		RemainingCount: int32(p.Remaining),
	}
	if p.EstimatedCompletion != nil {
		progress.EstimatedCompletionAt = siteapi.NewOptNilTimestamp(siteapi.Timestamp(*p.EstimatedCompletion))
	}
	res.Progress = siteapi.NewOptNilSiteBroadcastProgress(progress)
	return res, nil
}

func (h *Handlers) SiteBroadcastsList(ctx context.Context, params siteapi.SiteBroadcastsListParams) (siteapi.SiteBroadcastsListRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
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

	q := ws.Broadcast().Query()

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
		if resources[i], err = broadcastResource(ctx, ws, b); err != nil {
			return nil, err
		}
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
	ws, err := h.scopedFor(ctx, params.Slug)
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

	q := ws.Broadcast().Create().
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
	ws, err := h.scopedFor(ctx, params.Slug)
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
	b, err := ws.Broadcast().Query().
		Where(broadcast.IDEQ(id)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsGetNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res, err := broadcastResource(ctx, ws, b)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

func (h *Handlers) SiteBroadcastsUpdate(ctx context.Context, req *siteapi.SiteUpdateBroadcastInput, params siteapi.SiteBroadcastsUpdateParams) (siteapi.SiteBroadcastsUpdateRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
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
	current, err := ws.Broadcast().Query().
		Where(broadcast.IDEQ(id)).
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

	q := ws.Broadcast().UpdateOneID(id).
		SetNillableName(convert.StringPtr(req.Name)).
		SetNillableSubject(convert.StringPtr(req.Subject)).
		SetNillableFromName(convert.StringPtr(req.FromName)).
		SetNillableFromEmail(convert.StringPtr(req.FromEmail)).
		SetNillableBody(convert.StringPtr(req.Body)).
		SetNillableSegmentID(segmentID).
		SetNillableIntegrationID(integrationID)
	// JSON Merge Patch: an explicit null clears the field, an absent key keeps it.
	if req.FromName.IsNull() {
		q = q.ClearFromName()
	}
	if req.FromEmail.IsNull() {
		q = q.ClearFromEmail()
	}
	if req.SegmentId.IsNull() {
		q = q.ClearSegmentID()
	}
	if req.IntegrationId.IsNull() {
		q = q.ClearIntegrationID()
	}
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
	ws, err := h.scopedFor(ctx, params.Slug)
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
	if _, err := ws.BroadcastRecipient().Delete().
		Where(broadcastrecipient.BroadcastID(id)).
		Exec(ctx); err != nil {
		return nil, err
	}
	err = ws.Broadcast().DeleteOneID(id).Exec(ctx)
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
	ws, err := h.scopedFor(ctx, params.Slug)
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
	res, err := broadcastResource(ctx, ws, b)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// SiteBroadcastsSchedule schedules a draft broadcast to send at a future time.
func (h *Handlers) SiteBroadcastsSchedule(ctx context.Context, req *siteapi.SiteScheduleBroadcastInput, params siteapi.SiteBroadcastsScheduleParams) (siteapi.SiteBroadcastsScheduleRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
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
	ws, err := h.scopedFor(ctx, params.Slug)
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
	b, err := ws.Broadcast().Query().
		Where(broadcast.IDEQ(id)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsTestSendNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	if detail := h.outbound.SendBroadcastTest(ctx, ws, b, string(req.Email)); detail != "" {
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, detail))
		return &v, nil
	}
	return &siteapi.SiteBroadcastsTestSendNoContent{}, nil
}
