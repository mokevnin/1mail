package site

import (
	"context"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/pagination"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) SiteContactsList(ctx context.Context, params siteapi.SiteContactsListParams) (siteapi.SiteContactsListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := h.ent.Contact.Query().Where(contact.WorkspaceID(ws))
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
		resources[i] = mapper.ContactToResource(c)
	}

	return &siteapi.SiteContactsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteContactsCreate(ctx context.Context, req *siteapi.SiteCreateContactInput, params siteapi.SiteContactsCreateParams) (siteapi.SiteContactsCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	// Create the contact and publish contact.created in one transaction
	// (transactional outbox): the event is committed iff the row is. The
	// "persist" subscriber writes the engagement-log Event row from the event;
	// it is no longer written inline here.
	var c *ent.Contact
	err = h.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		q := tx.Contact.Create().
			SetWorkspaceID(ws).
			SetEmail(string(req.Email)).
			SetNillableFirstName(convert.StringPtr(req.FirstName)).
			SetNillableLastName(convert.StringPtr(req.LastName)).
			SetNillableTimeZone(convert.StringPtr(req.TimeZone))
		if v, ok := req.CustomFields.Get(); ok {
			q = q.SetCustomFields(map[string]string(v))
		}
		created, err := q.Save(ctx)
		if err != nil {
			return err
		}
		c = created
		return pub.Publish(ctx, events.ContactCreated{WorkspaceID: ws, ContactID: c.ID, Email: c.Email})
	})
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteContactsCreateConflict(problemWithErrors(http.StatusConflict, "email already exists", map[string][]string{
			"email": {"email already exists"},
		}))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	// The persist + automations subscribers handle the engagement log and
	// enrollment off the published contact.created event — no direct calls here.
	res := mapper.ContactToResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsGet(ctx context.Context, params siteapi.SiteContactsGetParams) (siteapi.SiteContactsGetRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsGetNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsGetBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	c, err := h.ent.Contact.Query().
		Where(contact.IDEQ(id), contact.WorkspaceID(ws)).
		Only(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsGetNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	res := mapper.ContactToResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsUpdate(ctx context.Context, req *siteapi.SiteUpdateContactInput, params siteapi.SiteContactsUpdateParams) (siteapi.SiteContactsUpdateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsUpdateBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
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
	res := mapper.ContactToResource(c)
	return &res, nil
}

func (h *Handlers) SiteContactsDelete(ctx context.Context, params siteapi.SiteContactsDeleteParams) (siteapi.SiteContactsDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteContactsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.Contact.DeleteOneID(id).Where(contact.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsDeleteNotFound(problem(http.StatusNotFound, "contact not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteContactsDeleteNoContent{}, nil
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
