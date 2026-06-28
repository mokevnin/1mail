package site

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/emailrender"
	"github.com/mokevnin/1mail/internal/messaging"
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
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	segmentID, ok := optEntityID(req.SegmentId)
	if !ok {
		v := siteapi.SiteBroadcastsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "invalid segmentId"))
		return &v, nil
	}
	integrationID, ok := optEntityID(req.IntegrationId)
	if !ok {
		v := siteapi.SiteBroadcastsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "invalid integrationId"))
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
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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
		v := siteapi.SiteBroadcastsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "invalid segmentId"))
		return &v, nil
	}
	integrationID, ok := optEntityID(req.IntegrationId)
	if !ok {
		v := siteapi.SiteBroadcastsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "invalid integrationId"))
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
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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

// SiteBroadcastsSend sends a draft broadcast immediately. The actual dispatch is
// performed asynchronously by the river worker (wired in a later step); here we
// validate the broadcast is sendable and move it into the sending state.
func (h *Handlers) SiteBroadcastsSend(ctx context.Context, params siteapi.SiteBroadcastsSendParams) (siteapi.SiteBroadcastsSendRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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

	b, err := h.ent.Broadcast.Query().
		Where(broadcast.IDEQ(id), broadcast.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsSendNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if b.Status != broadcast.StatusDraft && b.Status != broadcast.StatusScheduled {
		v := siteapi.SiteBroadcastsSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "broadcast is already sending or sent"))
		return &v, nil
	}

	// Enqueue first, then flip status: if the enqueue fails we don't strand the
	// broadcast in "sending" with no job behind it.
	if err := h.enqueuer.EnqueueBroadcast(ctx, b.ID, nil); err != nil {
		return nil, err
	}
	b, err = b.Update().SetStatus(broadcast.StatusSending).ClearScheduledAt().Save(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

// SiteBroadcastsSchedule schedules a draft broadcast to send at a future time.
func (h *Handlers) SiteBroadcastsSchedule(ctx context.Context, req *siteapi.SiteScheduleBroadcastInput, params siteapi.SiteBroadcastsScheduleParams) (siteapi.SiteBroadcastsScheduleRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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

	b, err := h.ent.Broadcast.Query().
		Where(broadcast.IDEQ(id), broadcast.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteBroadcastsScheduleNotFound(problem(http.StatusNotFound, "broadcast not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if b.Status != broadcast.StatusDraft && b.Status != broadcast.StatusScheduled {
		v := siteapi.SiteBroadcastsScheduleUnprocessableEntity(problem(http.StatusUnprocessableEntity, "broadcast is already sending or sent"))
		return &v, nil
	}

	when := time.Time(req.ScheduledAt)
	if err := h.enqueuer.EnqueueBroadcast(ctx, b.ID, &when); err != nil {
		return nil, err
	}
	b, err = b.Update().
		SetStatus(broadcast.StatusScheduled).
		SetScheduledAt(when).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.BroadcastToResource(b)
	return &res, nil
}

// SiteBroadcastsTestSend renders the broadcast with sample merge data and sends
// it to a single address — no recipient rows, no tracking (a token would point
// at a nonexistent recipient). Used to preview the rendered email.
func (h *Handlers) SiteBroadcastsTestSend(ctx context.Context, req *siteapi.SiteTestSendBroadcastInput, params siteapi.SiteBroadcastsTestSendParams) (siteapi.SiteBroadcastsTestSendRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
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

	sender, err := messaging.NewResolver(h.ent, h.cipher, h.catalog).EmailSender(ctx, ws)
	if err != nil {
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "no sending integration configured"))
		return &v, nil
	}

	to := string(req.Email)
	bindings := map[string]any{"first_name": "Alex", "last_name": "Sample", "email": to}
	email, rerr := emailrender.RenderEmail(b.Subject, b.Body, bindings)
	if rerr != nil {
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, rerr.Error()))
		return &v, nil
	}

	var fromEmail, fromName string
	if b.FromEmail != nil {
		fromEmail = *b.FromEmail
	}
	if b.FromName != nil {
		fromName = *b.FromName
	}
	if err := sender.Send(ctx, messaging.EmailMessage{
		From:     fromEmail,
		FromName: fromName,
		To:       to,
		Subject:  "[Test] " + email.Subject,
		HTML:     email.HTML,
		Text:     email.Text,
	}); err != nil {
		v := siteapi.SiteBroadcastsTestSendUnprocessableEntity(problem(http.StatusUnprocessableEntity, "send failed: "+err.Error()))
		return &v, nil
	}
	return &siteapi.SiteBroadcastsTestSendNoContent{}, nil
}
