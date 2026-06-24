package external

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) ContactsList(ctx context.Context, params externalapi.ContactsListParams) (externalapi.ContactsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := h.ent.Contact.Query().Where(contact.WorkspaceID(ws))
	if status, ok := params.Status.Get(); ok {
		q = q.Where(contact.StatusEQ(contact.Status(string(status))))
	}

	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}

	items, err := q.Order(ent.Asc(contact.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]externalapi.ContactResource, len(items))
	for i, c := range items {
		resources[i] = contactResource(c)
	}

	return &externalapi.ContactsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) ContactsCreate(ctx context.Context, req *externalapi.CreateContactInput) (externalapi.ContactsCreateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.ContactsCreateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	q := h.ent.Contact.Create().
		SetWorkspaceID(auth.WorkspaceID(auth.GetTokenAuth(ctx))).
		SetEmail(string(req.Email)).
		SetNillableFirstName(convert.StringPtr(req.FirstName)).
		SetNillableLastName(convert.StringPtr(req.LastName)).
		SetNillableTimeZone(convert.StringPtr(req.TimeZone))
	if v, ok := req.CustomFields.Get(); ok {
		q = q.SetCustomFields(map[string]string(v))
	}

	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		res := externalapi.ContactsCreateConflict(problem(http.StatusConflict, "email already exists"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	resource := contactResource(c)
	return &resource, nil
}

func (h *Handlers) ContactsGet(ctx context.Context, params externalapi.ContactsGetParams) (externalapi.ContactsGetRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsGetUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.ContactsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	c, err := h.ent.Contact.Query().
		Where(contact.IDEQ(id), contact.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.ContactsGetNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	resource := contactResource(c)
	return &resource, nil
}

func (h *Handlers) ContactsUpdate(ctx context.Context, req *externalapi.UpdateContactInput, params externalapi.ContactsUpdateParams) (externalapi.ContactsUpdateRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.ContactsUpdateUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.ContactsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	q := h.ent.Contact.UpdateOneID(id).
		Where(contact.WorkspaceID(ws)).
		SetNillableFirstName(convert.StringPtr(req.FirstName)).
		SetNillableLastName(convert.StringPtr(req.LastName)).
		SetNillableTimeZone(convert.StringPtr(req.TimeZone))
	if v, ok := req.CustomFields.Get(); ok {
		q = q.SetCustomFields(map[string]string(v))
	}

	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		res := externalapi.ContactsUpdateConflict(problem(http.StatusConflict, "email already exists"))
		return &res, nil
	}
	if ent.IsNotFound(err) {
		res := externalapi.ContactsUpdateNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	resource := contactResource(c)
	return &resource, nil
}

func (h *Handlers) ContactsDelete(ctx context.Context, params externalapi.ContactsDeleteParams) (externalapi.ContactsDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.ContactsDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.ContactsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	err = h.ent.Contact.DeleteOneID(id).Where(contact.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.ContactsDeleteNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.ContactsDeleteNoContent{}, nil
}
