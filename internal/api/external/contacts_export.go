package external

import (
	"context"
	"errors"
	"net/http"

	externalapi "github.com/mokevnin/1mail/gen/external"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/contactexport"
	"github.com/samber/lo"
)

// ContactsExport streams the subject-access Data export of one contact (ADR 0021),
// needing only the contact read scope.
func (h *Handlers) ContactsExport(ctx context.Context, params externalapi.ContactsExportParams) (externalapi.ContactsExportRes, error) {
	if !auth.HasScope(auth.GetTokenAuth(ctx), "contacts:read") {
		res := externalapi.ContactsExportUnauthorized(problem(http.StatusUnauthorized, "insufficient scope"))
		return &res, nil
	}

	var id *int64
	if v, ok := params.ID.Get(); ok {
		parsed, err := parseEntityID(v)
		if err != nil {
			res := externalapi.ContactsExportBadRequest(problem(http.StatusBadRequest, "invalid id"))
			return &res, nil
		}
		id = &parsed
	}
	var email *string
	if v, ok := params.Email.Get(); ok {
		email = lo.ToPtr(string(v))
	}

	s := auth.TokenScoped(ctx)
	c, err := contactexport.Find(ctx, s, id, email)
	if errors.Is(err, contactexport.ErrIdentifier) {
		res := externalapi.ContactsExportBadRequest(problem(http.StatusBadRequest, "pass exactly one of id or email"))
		return &res, nil
	}
	if errors.Is(err, contactexport.ErrNotFound) {
		res := externalapi.ContactsExportNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	return &externalapi.ContactsExportOKHeaders{
		ContentDisposition: `attachment; filename="` + contactexport.Filename(c) + `"`,
		Response:           externalapi.ContactsExportOK{Data: contactexport.Open(ctx, s, c)},
	}, nil
}
