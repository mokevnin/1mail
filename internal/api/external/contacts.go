package external

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) ContactsList(ctx context.Context, params externalapi.ContactsListParams) (externalapi.ContactsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	page, pageSize := pagination.Normalize(optInt32Ptr(params.Page), optInt32Ptr(params.PageSize))

	q := h.ent.Contact.Query()
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
		SetEmail(string(req.Email)).
		SetNillableFirstName(optNilStringPtr(req.FirstName)).
		SetNillableLastName(optNilStringPtr(req.LastName)).
		SetNillableTimeZone(optNilTimeZonePtr(req.TimeZone))
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

	c, err := h.ent.Contact.Get(ctx, id)
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

	q := h.ent.Contact.UpdateOneID(id).
		SetNillableFirstName(optNilStringPtr(req.FirstName)).
		SetNillableLastName(optNilStringPtr(req.LastName)).
		SetNillableTimeZone(optNilTimeZonePtr(req.TimeZone))
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

	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		res := externalapi.ContactsDeleteNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.ContactsDeleteNoContent{}, nil
}
