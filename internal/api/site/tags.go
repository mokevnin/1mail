package site

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/samber/lo"

	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/tags"
)

func (h *Handlers) SiteTagsList(ctx context.Context, params siteapi.SiteTagsListParams) (siteapi.SiteTagsListRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTagsListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	all, err := h.tags.List(ctx, ws)
	if err != nil {
		return nil, err
	}
	page := siteTagsPage(all, params.Page, params.PageSize)
	return &siteapi.SiteTagsListOK{
		Items:      page.Items,
		Page:       int32(page.Page),
		PageSize:   int32(page.PageSize),
		TotalItems: int32(page.TotalItems),
		TotalPages: int32(page.TotalPages),
	}, nil
}

func (h *Handlers) SiteTagsListForContact(ctx context.Context, params siteapi.SiteTagsListForContactParams) (siteapi.SiteTagsListForContactRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTagsListForContactNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(string(params.ContactId), 10, 64)
	if err != nil {
		v := siteapi.SiteTagsListForContactBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	mine, err := h.tags.ForContact(ctx, ws, id)
	if errors.Is(err, tags.ErrContactNotFound) {
		v := siteapi.SiteTagsListForContactNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	page := siteTagsPage(mine, params.Page, params.PageSize)
	return &siteapi.SiteTagsListForContactOK{
		Items:      page.Items,
		Page:       int32(page.Page),
		PageSize:   int32(page.PageSize),
		TotalItems: int32(page.TotalItems),
		TotalPages: int32(page.TotalPages),
	}, nil
}

func (h *Handlers) SiteTagsApply(ctx context.Context, req *siteapi.SiteApplyTagInput, params siteapi.SiteTagsApplyParams) (siteapi.SiteTagsApplyRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTagsApplyNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(string(params.ContactId), 10, 64)
	if err != nil {
		v := siteapi.SiteTagsApplyBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	t, err := h.tags.Apply(ctx, ws, id, req.Name)
	if errors.Is(err, tags.ErrContactNotFound) {
		v := siteapi.SiteTagsApplyNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if errors.Is(err, tags.ErrInvalidName) {
		v := siteapi.SiteTagsApplyUnprocessableEntity(problem(http.StatusUnprocessableEntity, "name is required"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	resource := mapper.TagToResource(t)
	return &resource, nil
}

func (h *Handlers) SiteTagsRemove(ctx context.Context, params siteapi.SiteTagsRemoveParams) (siteapi.SiteTagsRemoveRes, error) {
	ws, err := h.workspaceID(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTagsRemoveNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	id, err := strconv.ParseInt(string(params.ContactId), 10, 64)
	if err != nil {
		v := siteapi.SiteTagsRemoveBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	err = h.tags.Remove(ctx, ws, id, params.Name)
	if errors.Is(err, tags.ErrContactNotFound) {
		v := siteapi.SiteTagsRemoveNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteTagsRemoveNoContent{}, nil
}

// siteTagsPage maps and pages a small in-memory tag set.
func siteTagsPage(items []*ent.Tag, page, pageSize siteapi.OptInt32) pagination.Page[siteapi.SiteTagResource] {
	resources := lo.Map(items, func(t *ent.Tag, _ int) siteapi.SiteTagResource { return mapper.TagToResource(t) })
	return pagination.Paginate(resources, convert.Ptr(page), convert.Ptr(pageSize))
}
