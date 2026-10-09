package site

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/segment"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/segments"
)

func (h *Handlers) SiteSegmentsList(ctx context.Context, params siteapi.SiteSegmentsListParams) (siteapi.SiteSegmentsListRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := ws.Segment().Query()

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	items, err := q.Order(ent.Asc(segment.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteSegmentResource, len(items))
	for i, s := range items {
		resources[i] = mapper.SegmentToResource(s)
	}

	return &siteapi.SiteSegmentsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteSegmentsCreate(ctx context.Context, req *siteapi.SiteCreateSegmentInput, params siteapi.SiteSegmentsCreateParams) (siteapi.SiteSegmentsCreateRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	s, err := h.segments.Create(ctx, ws, segments.CreateInput{
		Name:       req.Name,
		Definition: &req.Definition,
	})
	if errors.Is(err, segments.ErrInvalidDefinition) {
		v := siteapi.SiteSegmentsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SiteSegmentsGet(ctx context.Context, params siteapi.SiteSegmentsGetParams) (siteapi.SiteSegmentsGetRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSegmentsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	s, err := ws.Segment().Get(ctx, id)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsGetNotFound(problem(http.StatusNotFound, "segment not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SiteSegmentsUpdate(ctx context.Context, req *siteapi.SiteUpdateSegmentInput, params siteapi.SiteSegmentsUpdateParams) (siteapi.SiteSegmentsUpdateRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSegmentsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	in := segments.UpdateInput{
		Name:       convert.StringPtr(req.Name),
		Definition: convert.StringPtr(req.Definition),
	}
	s, err := h.segments.Update(ctx, ws, id, in)
	if errors.Is(err, segments.ErrInvalidDefinition) {
		v := siteapi.SiteSegmentsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &v, nil
	}
	if errors.Is(err, segments.ErrNotFound) {
		v := siteapi.SiteSegmentsUpdateNotFound(problem(http.StatusNotFound, "segment not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SiteSegmentsDelete(ctx context.Context, params siteapi.SiteSegmentsDeleteParams) (siteapi.SiteSegmentsDeleteRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSegmentsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = ws.Segment().DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsDeleteNotFound(problem(http.StatusNotFound, "segment not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSegmentsDeleteNoContent{}, nil
}

// SiteSegmentsPreview reports how many *deliverable* (active) contacts match a
// rule definition. It applies the same active filter the broadcast send path
// uses, so the previewed number matches what a broadcast to this segment ships.
func (h *Handlers) SiteSegmentsPreview(ctx context.Context, req *siteapi.SitePreviewSegmentInput, params siteapi.SiteSegmentsPreviewParams) (siteapi.SiteSegmentsPreviewRes, error) {
	ws, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSegmentsPreviewNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	def := ""
	if v := convert.StringPtr(req.Definition); v != nil {
		def = *v
	}
	count, err := h.segments.Preview(ctx, ws, def)
	if errors.Is(err, segments.ErrInvalidDefinition) {
		v := siteapi.SiteSegmentsPreviewUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SitePreviewSegmentResult{Count: int32(count)}, nil
}
