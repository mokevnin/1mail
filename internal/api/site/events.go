package site

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-faster/jx"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pagination"
)

func toEventResource(e *ent.Event) siteapi.SiteEventResource {
	res := siteapi.SiteEventResource{
		ID:        siteapi.EntityId(strconv.FormatInt(e.ID, 10)),
		SubjectId: e.SubjectID,
		Action:    e.Action,
		CreatedAt: siteapi.Timestamp(e.CreatedAt),
	}
	if e.Email != nil {
		res.Email = siteapi.NewOptNilString(*e.Email)
	}
	if e.OccurredAt != nil {
		res.OccurredAt = siteapi.NewOptNilTimestamp(siteapi.Timestamp(*e.OccurredAt))
	}
	if len(e.Properties) > 0 {
		props := make(siteapi.SiteEventResourceProperties, len(e.Properties))
		for k, v := range e.Properties {
			b, err := json.Marshal(v)
			if err != nil {
				continue
			}
			props[k] = jx.Raw(b)
		}
		res.Properties = siteapi.NewOptNilSiteEventResourceProperties(props)
	}
	return res
}

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
		resources[i] = toEventResource(e)
	}

	return &siteapi.SiteEventsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}
