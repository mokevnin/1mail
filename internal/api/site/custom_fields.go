package site

import (
	"context"
	"net/http"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/customfield"
	siteapi "github.com/mokevnin/1mail/gen/site"
)

// SiteCustomFieldsList returns the workspace's Custom field definitions — the typed,
// named attribute catalogue (ADR 0006) that feeds the segment builder. Definitions
// are auto-created on first sight at ingest; this is a read-only catalogue. The full
// set is returned in a single page (the catalogue is small and unpaginated).
func (h *Handlers) SiteCustomFieldsList(ctx context.Context, params siteapi.SiteCustomFieldsListParams) (siteapi.SiteCustomFieldsListRes, error) {
	ws, err := h.workspaceID(ctx, params.WorkspaceSlug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteCustomFieldsListNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	items, err := h.ent.CustomField.Query().
		Where(customfield.WorkspaceID(ws)).
		Order(ent.Asc(customfield.FieldKey)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteCustomFieldResource, len(items))
	for i, f := range items {
		resources[i] = mapper.CustomFieldToResource(f)
	}
	return &siteapi.SiteCustomFieldsListOK{
		Items:      resources,
		Page:       1,
		PageSize:   int32(len(resources)),
		TotalItems: int32(len(resources)),
		TotalPages: 1,
	}, nil
}
