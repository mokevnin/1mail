package external

import (
	"context"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/api/problems"
	"github.com/mokevnin/1mail/internal/api/resources"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

func (h *Handlers) ContactsList(ctx context.Context, req externalapi.ContactsListRequestObject) (externalapi.ContactsListResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "contacts:read") {
		return externalapi.ContactsList401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
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

	contactResources := lo.Map(items, func(c *ent.Contact, _ int) externalapi.ContactResource {
		return resources.ExternalContact(c)
	})

	return externalapi.ContactsList200JSONResponse{
		Items:      contactResources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) ContactsCreate(ctx context.Context, req externalapi.ContactsCreateRequestObject) (externalapi.ContactsCreateResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsCreate401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	q := h.ent.Contact.Create().
		SetEmail(string(req.Body.Email)).
		SetNillableFirstName(req.Body.FirstName).
		SetNillableLastName(req.Body.LastName).
		SetNillableTimeZone(req.Body.TimeZone)
	if req.Body.CustomFields != nil {
		q = q.SetCustomFields(*req.Body.CustomFields)
	}
	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		return externalapi.ContactsCreate409ApplicationProblemPlusJSONResponse(problems.ConflictWithErrors("email already exists", problems.FieldErrors{
			"email": {"email already exists"},
		}).External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsCreate201JSONResponse(resources.ExternalContact(c)), nil
}

func (h *Handlers) ContactsGet(ctx context.Context, req externalapi.ContactsGetRequestObject) (externalapi.ContactsGetResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "contacts:read") {
		return externalapi.ContactsGet401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsGet400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").External()), nil
	}
	c, err := h.ent.Contact.Get(ctx, id)
	if ent.IsNotFound(err) {
		return externalapi.ContactsGet404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsGet200JSONResponse(resources.ExternalContact(c)), nil
}

func (h *Handlers) ContactsUpdate(ctx context.Context, req externalapi.ContactsUpdateRequestObject) (externalapi.ContactsUpdateResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsUpdate401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsUpdate400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").External()), nil
	}
	q := h.ent.Contact.UpdateOneID(id).
		SetNillableFirstName(req.Body.FirstName).
		SetNillableLastName(req.Body.LastName).
		SetNillableTimeZone(req.Body.TimeZone)
	if req.Body.CustomFields != nil {
		q = q.SetCustomFields(*req.Body.CustomFields)
	}
	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		return externalapi.ContactsUpdate409ApplicationProblemPlusJSONResponse(problems.Conflict("email already exists").External()), nil
	}
	if ent.IsNotFound(err) {
		return externalapi.ContactsUpdate404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsUpdate200JSONResponse(resources.ExternalContact(c)), nil
}

func (h *Handlers) ContactsDelete(ctx context.Context, req externalapi.ContactsDeleteRequestObject) (externalapi.ContactsDeleteResponseObject, error) {
	if !apiauth.HasScope(apiauth.GetTokenAuth(ctx), "contacts:write") {
		return externalapi.ContactsDelete401ApplicationProblemPlusJSONResponse(problems.Unauthorized("insufficient scope").External()), nil
	}

	id, err := strconv.ParseInt(string(req.Id), 10, 64)
	if err != nil {
		return externalapi.ContactsDelete400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").External()), nil
	}
	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		return externalapi.ContactsDelete404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").External()), nil
	}
	if err != nil {
		return nil, err
	}
	return externalapi.ContactsDelete204Response{}, nil
}
