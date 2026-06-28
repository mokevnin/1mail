package site

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/emailtemplate"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) SiteTemplatesList(ctx context.Context, params siteapi.SiteTemplatesListParams) (siteapi.SiteTemplatesListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := h.ent.EmailTemplate.Query().Where(emailtemplate.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(emailtemplate.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteEmailTemplateResource, len(items))
	for i, tpl := range items {
		resources[i] = mapper.EmailTemplateToResource(tpl)
	}
	return &siteapi.SiteTemplatesListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteTemplatesCreate(ctx context.Context, req *siteapi.SiteCreateEmailTemplateInput, params siteapi.SiteTemplatesCreateParams) (siteapi.SiteTemplatesCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	q := h.ent.EmailTemplate.Create().SetWorkspaceID(ws).SetName(req.Name)
	if v, ok := req.Subject.Get(); ok {
		q = q.SetSubject(v)
	}
	if v, ok := req.BodyHtml.Get(); ok {
		q = q.SetBodyHTML(v)
	}
	if v, ok := req.BodyFormat.Get(); ok {
		q = q.SetBodyFormat(emailtemplate.BodyFormat(v))
	}
	tpl, err := q.Save(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) SiteTemplatesGet(ctx context.Context, params siteapi.SiteTemplatesGetParams) (siteapi.SiteTemplatesGetRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteTemplatesGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	tpl, err := h.ent.EmailTemplate.Query().
		Where(emailtemplate.IDEQ(id), emailtemplate.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesGetNotFound(problem(http.StatusNotFound, "template not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) SiteTemplatesUpdate(ctx context.Context, req *siteapi.SiteUpdateEmailTemplateInput, params siteapi.SiteTemplatesUpdateParams) (siteapi.SiteTemplatesUpdateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteTemplatesUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}

	q := h.ent.EmailTemplate.UpdateOneID(id).
		Where(emailtemplate.WorkspaceID(ws)).
		SetNillableName(convert.StringPtr(req.Name)).
		SetNillableSubject(convert.StringPtr(req.Subject)).
		SetNillableBodyHTML(convert.StringPtr(req.BodyHtml))
	if v, ok := req.BodyFormat.Get(); ok {
		q = q.SetBodyFormat(emailtemplate.BodyFormat(v))
	}
	tpl, err := q.Save(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesUpdateNotFound(problem(http.StatusNotFound, "template not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) SiteTemplatesDelete(ctx context.Context, params siteapi.SiteTemplatesDeleteParams) (siteapi.SiteTemplatesDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteTemplatesDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.EmailTemplate.DeleteOneID(id).Where(emailtemplate.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteTemplatesDeleteNotFound(problem(http.StatusNotFound, "template not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteTemplatesDeleteNoContent{}, nil
}
