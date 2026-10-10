package site

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/contactexport"
	"github.com/samber/lo"
)

// SiteContactsExport streams the subject-access Data export of one contact (ADR
// 0021). Any Workspace member may export: it needs contact read access only.
func (h *Handlers) SiteContactsExport(ctx context.Context, params siteapi.SiteContactsExportParams) (siteapi.SiteContactsExportRes, error) {
	scoped, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteContactsExportNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	var id *int64
	if v, ok := params.ID.Get(); ok {
		parsed, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			res := siteapi.SiteContactsExportBadRequest(problem(http.StatusBadRequest, "invalid id"))
			return &res, nil
		}
		id = &parsed
	}
	var email *string
	if v, ok := params.Email.Get(); ok {
		email = lo.ToPtr(string(v))
	}

	c, err := contactexport.Find(ctx, scoped, id, email)
	if errors.Is(err, contactexport.ErrIdentifier) {
		res := siteapi.SiteContactsExportBadRequest(problem(http.StatusBadRequest, "pass exactly one of id or email"))
		return &res, nil
	}
	if errors.Is(err, contactexport.ErrNotFound) {
		res := siteapi.SiteContactsExportNotFound(problem(http.StatusNotFound, "contact not found"))
		return &res, nil
	}
	if err != nil {
		return nil, err
	}

	return &siteapi.SiteContactsExportOKHeaders{
		ContentDisposition: `attachment; filename="` + contactexport.Filename(c) + `"`,
		Response:           siteapi.SiteContactsExportOK{Data: contactexport.Open(ctx, scoped, c)},
	}, nil
}
