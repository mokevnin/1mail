package external

import (
	"context"
	"errors"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/contacts"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) ContactsList(ctx context.Context, params externalapi.ContactsListParams) (externalapi.ContactsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := h.ent.Contact.Query().Where(contact.WorkspaceID(ws))

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
		resources[i] = mapper.ContactToResource(c)
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

	ws := auth.WorkspaceID(auth.GetTokenAuth(ctx))
	c, err := h.contacts.Create(ctx, ws, createContactAttributes(req))
	var conflict *contacts.ConflictError
	if errors.As(err, &conflict) {
		res := externalapi.ContactsCreateConflict(conflictProblem(conflict))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	resource := mapper.ContactToResource(c)
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

	resource := mapper.ContactToResource(c)
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
	c, err := h.contacts.Update(ctx, ws, id, updateContactAttributes(req))
	var conflict *contacts.ConflictError
	if errors.As(err, &conflict) {
		res := externalapi.ContactsUpdateConflict(conflictProblem(conflict))
		return &res, nil
	}
	if errors.Is(err, contacts.ErrNotFound) {
		res := externalapi.ContactsUpdateNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	resource := mapper.ContactToResource(c)
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

// conflictProblem renders a contacts.ConflictError, with the field-level error.
func conflictProblem(c *contacts.ConflictError) externalapi.ProblemDetails {
	p := problem(http.StatusConflict, c.Message())
	p.Errors = externalapi.NewOptProblemDetailsErrors(externalapi.ProblemDetailsErrors{c.Field: {c.Message()}})
	return p
}

func createContactAttributes(req *externalapi.CreateContactInput) contacts.Attributes {
	attrs := contacts.Attributes{
		SubjectID: convert.StringPtr(req.SubjectId),
		Email:     convert.StringPtr(req.Email),
		Phone:     convert.StringPtr(req.Phone),
		FirstName: convert.StringPtr(req.FirstName),
		LastName:  convert.StringPtr(req.LastName),
		TimeZone:  convert.StringPtr(req.TimeZone),
	}
	if v, ok := req.CustomFields.Get(); ok {
		attrs.CustomFields = convert.RawMap(v)
	}
	return attrs
}

func updateContactAttributes(req *externalapi.UpdateContactInput) contacts.Attributes {
	attrs := contacts.Attributes{
		SubjectID: convert.StringPtr(req.SubjectId),
		Email:     convert.StringPtr(req.Email),
		Phone:     convert.StringPtr(req.Phone),
		FirstName: convert.StringPtr(req.FirstName),
		LastName:  convert.StringPtr(req.LastName),
		TimeZone:  convert.StringPtr(req.TimeZone),
	}
	if v, ok := req.CustomFields.Get(); ok {
		attrs.CustomFields = convert.RawMap(v)
	}
	return attrs
}
