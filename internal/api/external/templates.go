package external

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/emailtemplate"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) TemplatesList(ctx context.Context, params externalapi.TemplatesListParams) (externalapi.TemplatesListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "templates:read") {
		res := externalapi.TemplatesListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := scoped.EmailTemplate().Query()
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Asc(emailtemplate.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]externalapi.TemplateResource, len(items))
	for i, tpl := range items {
		resources[i] = mapper.EmailTemplateToResource(tpl)
	}
	return &externalapi.TemplatesListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) TemplatesCreate(ctx context.Context, req *externalapi.CreateTemplateInput) (externalapi.TemplatesCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "templates:write") {
		res := externalapi.TemplatesCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	tpl, err := scoped.EmailTemplate().Create().
		SetName(req.Name).
		SetNillableSubject(convert.StringPtr(req.Subject)).
		SetNillableBody(convert.StringPtr(req.Body)).
		Save(ctx)
	if ent.IsValidationError(err) {
		res := externalapi.TemplatesCreateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "name must not be empty"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) TemplatesGet(ctx context.Context, params externalapi.TemplatesGetParams) (externalapi.TemplatesGetRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "templates:read") {
		res := externalapi.TemplatesGetUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.TemplatesGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	tpl, err := scoped.EmailTemplate().Get(ctx, id)
	if ent.IsNotFound(err) {
		res := externalapi.TemplatesGetNotFound(problem(http.StatusNotFound, "template not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) TemplatesUpdate(ctx context.Context, req *externalapi.UpdateTemplateInput, params externalapi.TemplatesUpdateParams) (externalapi.TemplatesUpdateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "templates:write") {
		res := externalapi.TemplatesUpdateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.TemplatesUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	tpl, err := scoped.EmailTemplate().UpdateOneID(id).
		SetNillableName(convert.StringPtr(req.Name)).
		SetNillableSubject(convert.StringPtr(req.Subject)).
		SetNillableBody(convert.StringPtr(req.Body)).
		Save(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.TemplatesUpdateNotFound(problem(http.StatusNotFound, "template not found"))
		return &res, nil
	}
	if ent.IsValidationError(err) {
		res := externalapi.TemplatesUpdateUnprocessableEntity(problem(http.StatusUnprocessableEntity, "name must not be empty"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.EmailTemplateToResource(tpl)
	return &res, nil
}

func (h *Handlers) TemplatesDelete(ctx context.Context, params externalapi.TemplatesDeleteParams) (externalapi.TemplatesDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "templates:write") {
		res := externalapi.TemplatesDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.TemplatesDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	scoped := auth.TokenScoped(ctx)
	err = scoped.EmailTemplate().DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.TemplatesDeleteNotFound(problem(http.StatusNotFound, "template not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.TemplatesDeleteNoContent{}, nil
}
