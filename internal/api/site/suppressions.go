package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/suppression"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/pagination"
)

// normalizeEmail lower-cases and trims an address so suppression lookups and
// upserts are case-insensitive (collect ingestion lower-cases event emails too).
func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func (h *Handlers) SiteSuppressionsList(ctx context.Context, params siteapi.SiteSuppressionsListParams) (siteapi.SiteSuppressionsListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSuppressionsListNotFound(problem(http.StatusNotFound, "workspace not found"))
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

	q := h.ent.Suppression.Query().Where(suppression.WorkspaceID(ws))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, err
	}
	items, err := q.Order(ent.Desc(suppression.FieldID)).
		Limit(pageSize).
		Offset(pagination.Offset(page, pageSize)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteSuppressionResource, len(items))
	for i, s := range items {
		resources[i] = mapper.SuppressionToResource(s)
	}
	return &siteapi.SiteSuppressionsListOK{
		Items:      resources,
		Page:       int32(page),
		PageSize:   int32(pageSize),
		TotalItems: int32(total),
		TotalPages: int32(pagination.TotalPages(total, pageSize)),
	}, nil
}

func (h *Handlers) SiteSuppressionsCreate(ctx context.Context, req *siteapi.SiteCreateSuppressionInput, params siteapi.SiteSuppressionsCreateParams) (siteapi.SiteSuppressionsCreateRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSuppressionsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	email := normalizeEmail(string(req.Email))
	if email == "" {
		v := siteapi.SiteSuppressionsCreateUnprocessableEntity(problemWithErrors(http.StatusUnprocessableEntity, "invalid email", map[string][]string{
			"email": {"must not be empty"},
		}))
		return &v, nil
	}

	// Manual suppression is idempotent per address: keep the existing entry (and
	// its reason) if the address is already suppressed.
	if err := h.ent.Suppression.Create().
		SetWorkspaceID(ws).
		SetEmail(email).
		SetReason(suppression.ReasonManual).
		OnConflictColumns(suppression.FieldWorkspaceID, suppression.FieldEmail).
		Ignore().
		Exec(ctx); err != nil {
		return nil, err
	}

	s, err := h.ent.Suppression.Query().
		Where(suppression.WorkspaceID(ws), suppression.EmailEQ(email)).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	res := mapper.SuppressionToResource(s)
	return &res, nil
}

func (h *Handlers) SiteSuppressionsDelete(ctx context.Context, params siteapi.SiteSuppressionsDeleteParams) (siteapi.SiteSuppressionsDeleteRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSuppressionsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteSuppressionsDeleteBadRequest(problem(http.StatusBadRequest, "invalid id"))
		return &v, nil
	}
	err = h.ent.Suppression.DeleteOneID(id).Where(suppression.WorkspaceID(ws)).Exec(ctx)
	if ent.IsNotFound(err) {
		v := siteapi.SiteSuppressionsDeleteNotFound(problem(http.StatusNotFound, "suppression not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSuppressionsDeleteNoContent{}, nil
}
