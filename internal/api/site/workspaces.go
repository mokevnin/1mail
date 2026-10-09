package site

import (
	"context"
	"net/http"
	"strings"

	"github.com/mokevnin/1mail/ent"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/i18n"
)

// SiteWorkspacesList returns the workspaces the authenticated user is a member of.
func (h *Handlers) SiteWorkspacesList(ctx context.Context) ([]siteapi.SiteWorkspaceResource, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return []siteapi.SiteWorkspaceResource{}, nil
	}

	items, err := h.accounts.WorkspacesOf(ctx, a.UserID)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteWorkspaceResource, len(items))
	for i, w := range items {
		resources[i] = mapper.WorkspaceToResource(w)
	}
	return resources, nil
}

// SiteWorkspacesUpdate renames a workspace owned by the authenticated user. The
// slug is immutable, so only the display name changes.
func (h *Handlers) SiteWorkspacesUpdate(ctx context.Context, req *siteapi.SiteUpdateWorkspaceInput, params siteapi.SiteWorkspacesUpdateParams) (siteapi.SiteWorkspacesUpdateRes, error) {
	s, err := h.scopedFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteWorkspacesUpdateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		v := siteapi.SiteWorkspacesUpdateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			i18n.T("errors.name_empty", nil),
			map[string][]string{"name": {i18n.T("errors.name_empty", nil)}},
		))
		return &v, nil
	}

	var postal *string
	// postalAddress is optional in the contract: absent = leave unchanged, present
	// (incl. empty string) = set/clear. Trimmed so a whitespace-only value clears it.
	if req.PostalAddress.Set {
		v := strings.TrimSpace(req.PostalAddress.Value)
		postal = &v
	}
	w, err := h.accounts.UpdateWorkspace(ctx, s, name, postal)
	if err != nil {
		return nil, err
	}
	resource := mapper.WorkspaceToResource(w)
	return &resource, nil
}
