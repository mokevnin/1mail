package external

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/contacts"
	"github.com/mokevnin/1mail/internal/convert"
	"github.com/mokevnin/1mail/internal/erasure"
	"github.com/mokevnin/1mail/internal/eventlog"
	"github.com/mokevnin/1mail/internal/pagination"
)

func (h *Handlers) ContactsList(ctx context.Context, params externalapi.ContactsListParams) (externalapi.ContactsListRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsListUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	page, pageSize := pagination.Normalize(convert.Ptr(params.Page), convert.Ptr(params.PageSize))

	q := auth.TokenScoped(ctx).Contact().Query()

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

	c, err := h.contacts.Create(ctx, auth.TokenScoped(ctx), createContactAttributes(req))
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

	c, err := auth.TokenScoped(ctx).Contact().Get(ctx, id)
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

	c, err := h.contacts.Update(ctx, auth.TokenScoped(ctx), id, updateContactAttributes(req))
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

// ContactsDelete is Erasure (ADR 0021): deleting a Contact removes its personal data.
// It has its own scope, separate from contacts:write, because it is irreversible.
func (h *Handlers) ContactsDelete(ctx context.Context, params externalapi.ContactsDeleteParams) (externalapi.ContactsDeleteRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:erase") {
		res := externalapi.ContactsDeleteUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	id, err := parseEntityID(params.ID)
	if err != nil {
		res := externalapi.ContactsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &res, nil
	}

	err = h.erasure.Erase(ctx, auth.TokenScoped(ctx), erasure.ByContactID(id))
	if errors.Is(err, erasure.ErrNotFound) {
		res := externalapi.ContactsDeleteNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}
	return &externalapi.ContactsDeleteNoContent{}, nil
}

// ContactsBatchUpsert upserts each Contact independently and reports a per-item result.
func (h *Handlers) ContactsBatchUpsert(ctx context.Context, req *externalapi.UpsertContactsInput) (externalapi.ContactsBatchUpsertRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:write") {
		res := externalapi.ContactsBatchUpsertUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	items := make([]contacts.Attributes, len(req.Contacts))
	for i, c := range req.Contacts {
		items[i] = contacts.Attributes{
			SubjectID: convert.StringPtr(c.SubjectId),
			Email:     convert.StringPtr(c.Email),
			Phone:     convert.StringPtr(c.Phone),
			FirstName: convert.StringPtr(c.FirstName),
			LastName:  convert.StringPtr(c.LastName),
			TimeZone:  convert.StringPtr(c.TimeZone),
		}
		if v, ok := c.CustomFields.Get(); ok {
			items[i].CustomFields = convert.RawMap(v)
		}
	}
	outcomes := h.contacts.UpsertBatch(ctx, auth.TokenScoped(ctx), items)

	results := make([]externalapi.ContactBatchItemResult, len(outcomes))
	for i, o := range outcomes {
		results[i].Index = int32(i)
		switch {
		case o.Err != nil:
			results[i].Status = externalapi.ContactBatchStatusFailed
			results[i].Error = externalapi.NewOptString(itemError(o.Err))
		case o.Result.Created:
			results[i].Status = externalapi.ContactBatchStatusCreated
		default:
			results[i].Status = externalapi.ContactBatchStatusUpdated
		}
		if o.Err == nil {
			results[i].ContactId = externalapi.NewOptEntityId(externalapi.EntityId(strconv.FormatInt(o.Result.Contact.ID, 10)))
		}
	}
	return &externalapi.UpsertContactsResult{Results: results}, nil
}

// itemError words a batch item's failure for the caller: domain errors as their own
// message, anything else (storage) as a generic one so internals do not leak.
func itemError(err error) string {
	var conflict *contacts.ConflictError
	switch {
	case errors.As(err, &conflict):
		return conflict.Message()
	case errors.Is(err, contacts.ErrIdentityRequired), errors.Is(err, eventlog.ErrInvalid):
		return err.Error()
	default:
		return "internal error"
	}
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
