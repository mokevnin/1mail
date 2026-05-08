package site

import (
	"context"
	"strconv"

	"github.com/mokevnin/1mail/backend/ent"
	"github.com/mokevnin/1mail/backend/ent/contact"
	siteapi "github.com/mokevnin/1mail/backend/gen/site"
	"github.com/mokevnin/1mail/backend/internal/pagination"
	"github.com/mokevnin/1mail/backend/internal/service"
)

type Handlers struct {
	ent *ent.Client
}

func NewHandlers(client *ent.Client) *Handlers {
	return &Handlers{ent: client}
}

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

	resources := make([]siteapi.SiteContactResource, len(items))
	for i, c := range items {
		resources[i] = toResource(c)
	}

	return siteapi.SiteContactsList200JSONResponse{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteContactsCreate(ctx context.Context, req siteapi.SiteContactsCreateRequestObject) (siteapi.SiteContactsCreateResponseObject, error) {
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
		return siteapi.SiteContactsCreate409ApplicationProblemPlusJSONResponse(conflict("email already exists")), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsCreate201JSONResponse(toResource(c)), nil
}

func (h *Handlers) SiteContactsGet(ctx context.Context, req siteapi.SiteContactsGetRequestObject) (siteapi.SiteContactsGetResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsGet400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}
	c, err := h.ent.Contact.Get(ctx, id)
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsGet404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsGet200JSONResponse(toResource(c)), nil
}

func (h *Handlers) SiteContactsUpdate(ctx context.Context, req siteapi.SiteContactsUpdateRequestObject) (siteapi.SiteContactsUpdateResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsUpdate400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
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
		return siteapi.SiteContactsUpdate409ApplicationProblemPlusJSONResponse(conflict("email already exists")), nil
	}
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsUpdate404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsUpdate200JSONResponse(toResource(c)), nil
}

func (h *Handlers) SiteContactsDelete(ctx context.Context, req siteapi.SiteContactsDeleteRequestObject) (siteapi.SiteContactsDeleteResponseObject, error) {
	id, err := strconv.ParseInt(req.Id, 10, 64)
	if err != nil {
		return siteapi.SiteContactsDelete400ApplicationProblemPlusJSONResponse(badRequest("invalid id")), nil
	}
	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		return siteapi.SiteContactsDelete404ApplicationProblemPlusJSONResponse(notFound("contact not found")), nil
	}
	if err != nil {
		return nil, err
	}
	return siteapi.SiteContactsDelete204Response{}, nil
}

func toResource(c *ent.Contact) siteapi.SiteContactResource {
	r := siteapi.SiteContactResource{
		Id:        strconv.FormatInt(c.ID, 10),
		Email:     siteapi.EmailAddress(c.Email),
		Status:    siteapi.SiteContactStatus(c.Status),
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		TimeZone:  c.TimeZone,
	}
	if c.CustomFields != nil {
		cf := map[string]string(c.CustomFields)
		r.CustomFields = &cf
	}
	return r
}

func ptr[T any](v T) *T { return &v }

func strptr(s string) *string { return &s }

func conflict(detail string) siteapi.ProblemDetails {
	return problem(409, "Conflict", detail)
}

func notFound(detail string) siteapi.ProblemDetails {
	return problem(404, "Not Found", detail)
}

func badRequest(detail string) siteapi.ProblemDetails {
	return problem(400, "Bad Request", detail)
}

func problem(status int32, title, detail string) siteapi.ProblemDetails {
	return siteapi.ProblemDetails{
		Status: &status,
		Title:  strptr(title),
		Detail: strptr(detail),
	}
}

// ensure Handlers implements the interface
var _ siteapi.StrictServerInterface = (*Handlers)(nil)

