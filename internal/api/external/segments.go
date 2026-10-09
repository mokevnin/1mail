package external

import (
	"context"
	"errors"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/segment"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/segments"
)

func (h *Handlers) SegmentsList(ctx context.Context, params externalapi.SegmentsListParams) (externalapi.SegmentsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:read") {
		res := externalapi.SegmentsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := h.ent.Segment.Query().Where(segment.WorkspaceID(ws))
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

	out := make([]externalapi.SegmentResource, len(items))
	for i, s := range items {
		out[i] = mapper.SegmentToResource(s)
	}
	return &externalapi.SegmentsListOK{
		Items:      out,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SegmentsCreate(ctx context.Context, req *externalapi.CreateSegmentInput) (externalapi.SegmentsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:write") {
		res := externalapi.SegmentsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	s, err := h.segments.Create(ctx, ws, segments.CreateInput{
		Name:       req.Name,
		Definition: &req.Definition,
	})
	if errors.Is(err, segments.ErrInvalidDefinition) {
		res := externalapi.SegmentsCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SegmentsGet(ctx context.Context, params externalapi.SegmentsGetParams) (externalapi.SegmentsGetRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:read") {
		res := externalapi.SegmentsGetUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.SegmentsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	s, err := h.ent.Segment.Query().Where(segment.ID(id), segment.WorkspaceID(ws)).Only(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.SegmentsGetNotFound(problem(http.StatusNotFound, "segment not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SegmentsUpdate(ctx context.Context, req *externalapi.UpdateSegmentInput, params externalapi.SegmentsUpdateParams) (externalapi.SegmentsUpdateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:write") {
		res := externalapi.SegmentsUpdateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.SegmentsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	in := segments.UpdateInput{
		Name:       convert.StringPtr(req.Name),
		Definition: convert.StringPtr(req.Definition),
	}
	s, err := h.segments.Update(ctx, ws, id, in)
	if errors.Is(err, segments.ErrInvalidDefinition) {
		res := externalapi.SegmentsUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &res, nil
	}
	if errors.Is(err, segments.ErrNotFound) {
		res := externalapi.SegmentsUpdateNotFound(problem(http.StatusNotFound, "segment not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.SegmentToResource(s)
	return &res, nil
}

func (h *Handlers) SegmentsDelete(ctx context.Context, params externalapi.SegmentsDeleteParams) (externalapi.SegmentsDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:write") {
		res := externalapi.SegmentsDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.SegmentsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}
	err = h.ent.Segment.DeleteOneID(id).Where(segment.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.SegmentsDeleteNotFound(problem(http.StatusNotFound, "segment not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.SegmentsDeleteNoContent{}, nil
}

func (h *Handlers) SegmentsPreview(ctx context.Context, req *externalapi.PreviewSegmentInput) (externalapi.SegmentsPreviewRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "segments:read") {
		res := externalapi.SegmentsPreviewUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	def := ""
	if v := convert.StringPtr(req.Definition); v != nil {
		def = *v
	}
	count, err := h.segments.Preview(ctx, ws, def)
	if errors.Is(err, segments.ErrInvalidDefinition) {
		res := externalapi.SegmentsPreviewUnprocessableEntity(problem(http.StatusUnprocessableEntity, err.Error()))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.PreviewSegmentResult{Count: int32(count)}, nil
}
