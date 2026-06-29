package site

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pagination"
)

// SiteEventsList returns the workspace's events, most recent first.
func (h *Handlers) SiteEventsList(ctx context.Context, params siteapi.SiteEventsListParams) (siteapi.SiteEventsListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
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

	q := h.ent.Event.Query().Where(event.WorkspaceID(ws))
	if v, ok := params.Action.Get(); ok && v != "" {
		q = q.Where(event.ActionEQ(v))
	}
	if v, ok := params.Email.Get(); ok && v != "" {
		// Case-insensitive: contact emails are stored as entered, but collect
		// ingestion lowercases event emails (service.normalizeLower), so an exact
		// match would miss a contact's tracked events.
		q = q.Where(event.EmailEqualFold(v))
	}
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	items, err := q.Order(ent.Desc(event.FieldCreatedAt), ent.Desc(event.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteEventResource, len(items))
	for i, e := range items {
		resources[i] = mapper.EventToResource(e)
	}

	return &siteapi.SiteEventsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

// SiteEventsActions returns the distinct event actions in the workspace, sorted —
// used to populate the segment builder's event-condition picker.
func (h *Handlers) SiteEventsActions(ctx context.Context, params siteapi.SiteEventsActionsParams) (siteapi.SiteEventsActionsRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	actions, err := h.ent.Event.Query().
		Where(event.WorkspaceID(ws)).
		Order(ent.Asc(event.FieldAction)).
		GroupBy(event.FieldAction).
		Strings(ctx)
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteEventActionsResult{Actions: actions}, nil
}
