package site

import (
	"context"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/problems"
	"github.com/mokevnin/1mail/internal/api/resources"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

func (h *Handlers) SiteContactsList(ctx context.Context, req siteapi.SiteContactsListRequestObject) (siteapi.SiteContactsListResponseObject, error) {
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

	contactResources := lo.Map(items, func(c *ent.Contact, _ int) siteapi.SiteContactResource {
		return resources.SiteContact(c)
	})

	return siteapi.SiteContactsList200JSONResponse{
		Items:      contactResources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteContactsCreate(ctx context.Context, req siteapi.SiteContactsCreateRequestObject) (siteapi.SiteContactsCreateResponseObject, error) {
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
		return siteapi.SiteContactsCreate409ApplicationProblemPlusJSONResponse(problems.ConflictWithErrors("email already exists", problems.FieldErrors{
			"email": {"email already exists"},
		}).Site()), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsCreate201JSONResponse(resources.SiteContact(c)), nil
}

func (h *Handlers) SiteContactsGet(ctx context.Context, req siteapi.SiteContactsGetRequestObject) (siteapi.SiteContactsGetResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsGet400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").Site()), nil
	}
	c, err := h.ent.Contact.Get(ctx, id)
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsGet404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").Site()), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsGet200JSONResponse(resources.SiteContact(c)), nil
}

func (h *Handlers) SiteContactsUpdate(ctx context.Context, req siteapi.SiteContactsUpdateRequestObject) (siteapi.SiteContactsUpdateResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsUpdate400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").Site()), nil
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
		return siteapi.SiteContactsUpdate409ApplicationProblemPlusJSONResponse(problems.Conflict("email already exists").Site()), nil
	}
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsUpdate404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").Site()), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsUpdate200JSONResponse(resources.SiteContact(c)), nil
}

func (h *Handlers) SiteContactsDelete(ctx context.Context, req siteapi.SiteContactsDeleteRequestObject) (siteapi.SiteContactsDeleteResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsDelete400ApplicationProblemPlusJSONResponse(problems.BadRequest("invalid id").Site()), nil
	}
	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsDelete404ApplicationProblemPlusJSONResponse(problems.NotFound("contact not found").Site()), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsDelete204Response{}, nil
}
