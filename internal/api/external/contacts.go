package external

import (
	"context"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

func (h *Handlers) ContactsList(ctx context.Context, req externalapi.ContactsListRequestObject) (externalapi.ContactsListResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "contacts:read") {
		return externalapi.ContactsList401ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	page, pageSize := pagination.Normalize(req.Params.Page, req.Params.PageSize)

	q := h.ent.Contact.Query()
	if req.Params.Status != nil {
		q = q.Where(contact.StatusEQ(contact.Status(string(*req.Params.Status))))
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

	resources := lo.Map(items, func(c *ent.Contact, _ int) externalapi.ContactResource {
		return toContactResource(c)
	})

	return externalapi.ContactsList200JSONResponse{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) ContactsCreate(ctx context.Context, req externalapi.ContactsCreateRequestObject) (externalapi.ContactsCreateResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsCreate401ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	var customFields map[string]string
	if req.Body.CustomFields != nil {
		customFields = *req.Body.CustomFields
	}
	c, err := service.CreateContact(ctx, h.ent, service.CreateContactInput{
		Email:        string(req.Body.Email),
		FirstName:    req.Body.FirstName,
		LastName:     req.Body.LastName,
		TimeZone:     req.Body.TimeZone,
		CustomFields: customFields,
	})
	if service.IsUniqueViolation(err) {
		return externalapi.ContactsCreate409ApplicationProblemPlusJSONResponse(conflictWithErrors("email already exists", fieldErrors{
			"email": {"email already exists"},
		})), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsCreate201JSONResponse(toContactResource(c)), nil
}

func (h *Handlers) ContactsGet(ctx context.Context, req externalapi.ContactsGetRequestObject) (externalapi.ContactsGetResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "contacts:read") {
		return externalapi.ContactsGet401ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsGet400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}
	c, err := h.ent.Contact.Get(ctx, id)
	if ent.IsNotFound(err) {
		return externalapi.ContactsGet404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsGet200JSONResponse(toContactResource(c)), nil
}

func (h *Handlers) ContactsUpdate(ctx context.Context, req externalapi.ContactsUpdateRequestObject) (externalapi.ContactsUpdateResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsUpdate401ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsUpdate400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}
	var customFields map[string]string
	hasCustomFields := req.Body.CustomFields != nil
	if hasCustomFields {
		customFields = *req.Body.CustomFields
	}
	c, err := service.UpdateContact(ctx, h.ent, id, service.UpdateContactInput{
		FirstName:       req.Body.FirstName,
		LastName:        req.Body.LastName,
		TimeZone:        req.Body.TimeZone,
		CustomFields:    customFields,
		HasCustomFields: hasCustomFields,
	})
	if service.IsUniqueViolation(err) {
		return externalapi.ContactsUpdate409ApplicationProblemPlusJSONResponse(conflict("email already exists")), nil
	}
	if ent.IsNotFound(err) {
		return externalapi.ContactsUpdate404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsUpdate200JSONResponse(toContactResource(c)), nil
}

func (h *Handlers) ContactsDelete(ctx context.Context, req externalapi.ContactsDeleteRequestObject) (externalapi.ContactsDeleteResponseObject, error) {
	if !hasScope(GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsDelete401ApplicationProblemPlusJSONResponse(forbidden("insufficient scope")), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsDelete400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}
	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		return externalapi.ContactsDelete404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsDelete204Response{}, nil
}
