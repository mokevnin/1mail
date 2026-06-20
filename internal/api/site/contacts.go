package site

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) SiteContactsList(ctx context.Context, params siteapi.SiteContactsListParams) (siteapi.SiteContactsListRes, error) {
	var pagePtr, pageSizePtr *int32
	if v, ok := params.Page.Get(); ok {
		pagePtr = &v
	}
	if v, ok := params.PageSize.Get(); ok {
		pageSizePtr = &v
	}
	page, pageSize := pagination.Normalize(pagePtr, pageSizePtr)

	q := h.ent.Contact.Query()
	if v, ok := params.Status.Get(); ok {
		q = q.Where(contact.StatusEQ(contact.Status(string(v))))
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
		resources[i] = toContactResource(c)
	}

	return &siteapi.SiteContactsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteContactsCreate(ctx context.Context, req *siteapi.SiteCreateContactInput) (siteapi.SiteContactsCreateRes, error) {
	q := h.ent.Contact.Create().
		SetEmail(string(req.Email)).
		SetNillableFirstName(optNilString(req.FirstName)).
		SetNillableLastName(optNilString(req.LastName)).
		SetNillableTimeZone(optNilTimeZone(req.TimeZone))
	if v, ok := req.CustomFields.Get(); ok {
		q = q.SetCustomFields(map[string]string(v))
	}
	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteContactsCreateConflict(problemWithErrors(http.StatusConflict, "email already exists", map[string][]string{
			"email": {"email already exists"},
		}))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := toContactResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsGet(ctx context.Context, params siteapi.SiteContactsGetParams) (siteapi.SiteContactsGetRes, error) {
	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	c, err := h.ent.Contact.Get(ctx, id)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsGetNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := toContactResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsUpdate(ctx context.Context, req *siteapi.SiteUpdateContactInput, params siteapi.SiteContactsUpdateParams) (siteapi.SiteContactsUpdateRes, error) {
	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	q := h.ent.Contact.UpdateOneID(id).
		SetNillableFirstName(optNilString(req.FirstName)).
		SetNillableLastName(optNilString(req.LastName)).
		SetNillableTimeZone(optNilTimeZone(req.TimeZone))
	if v, ok := req.CustomFields.Get(); ok {
		q = q.SetCustomFields(map[string]string(v))
	}
	c, err := q.Save(ctx)
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteContactsUpdateConflict(problem(http.StatusConflict, "email already exists"))
		return &v, nil
	}
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsUpdateNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := toContactResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsDelete(ctx context.Context, params siteapi.SiteContactsDeleteParams) (siteapi.SiteContactsDeleteRes, error) {
	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.Contact.DeleteOneID(id).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsDeleteNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteContactsDeleteNoContent{}, nil
}

// toContactResource maps an ent.Contact to the site API resource representation.
func toContactResource(c *ent.Contact) siteapi.SiteContactResource {
	res := siteapi.SiteContactResource{
		ID:        siteapi.EntityId(strconv.FormatInt(c.ID, 10)),
		Email:     siteapi.EmailAddress(c.Email),
		Status:    siteapi.SiteContactStatus(string(c.Status)),
		CreatedAt: siteapi.Timestamp(c.CreatedAt),
		UpdatedAt: siteapi.Timestamp(c.UpdatedAt),
	}
	if c.FirstName != nil {
		res.FirstName = siteapi.NewOptNilString(*c.FirstName)
	}
	if c.LastName != nil {
		res.LastName = siteapi.NewOptNilString(*c.LastName)
	}
	if c.TimeZone != nil {
		res.TimeZone = siteapi.NewOptNilTimeZoneName(siteapi.TimeZoneName(*c.TimeZone))
	}
	if c.CustomFields != nil {
		res.CustomFields = siteapi.NewOptNilSiteContactResourceCustomFields(siteapi.SiteContactResourceCustomFields(c.CustomFields))
	}
	return res
}

// optNilString converts an optional nullable string input into a *string for ent setters.
func optNilString(o siteapi.OptNilString) *string {
	if v, ok := o.Get(); ok {
		return &v
	}
	return nil
}

// optNilTimeZone converts an optional nullable time zone input into a *string for ent setters.
func optNilTimeZone(o siteapi.OptNilTimeZoneName) *string {
	if v, ok := o.Get(); ok {
		s := string(v)
		return &s
	}
	return nil
}

// problem builds a ProblemDetails with status, title and detail.
func problem(code int, detail string) siteapi.ProblemDetails {
	return siteapi.ProblemDetails{
		Status: siteapi.NewOptInt32(int32(code)),
		Title:  siteapi.NewOptString(http.StatusText(code)),
		Detail: siteapi.NewOptString(detail),
	}
}

// problemWithErrors builds a ProblemDetails with field-level validation errors.
func problemWithErrors(code int, detail string, errors map[string][]string) siteapi.ProblemDetails {
	p := problem(code, detail)
	p.Errors = siteapi.NewOptProblemDetailsErrors(siteapi.ProblemDetailsErrors(errors))
	return p
}
