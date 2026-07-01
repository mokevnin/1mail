package site

import (
	"context"
	"net/http"
	"strings"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/membership"
	"github.com/mokevnin/1mail/ent/workspace"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
)

// SiteWorkspacesList returns the workspaces the authenticated user is a member of.
func (h *Handlers) SiteWorkspacesList(ctx context.Context) ([]siteapi.SiteWorkspaceResource, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return []siteapi.SiteWorkspaceResource{}, nil
	}

	items, err := h.ent.Workspace.Query().
		Where(workspace.HasMembershipsWith(membership.UserID(a.UserID))).
		Order(ent.Asc(workspace.FieldID)).
		All(ctx)
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
	id, err := h.workspaceID(ctx, params.Slug)
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
			"name must not be empty",
			map[string][]string{"name": {"name must not be empty"}},
		))
		return &v, nil
	}

	w, err := h.ent.Workspace.UpdateOneID(id).SetName(name).Save(ctx)
	if err != nil {
		return nil, err
	}
	resource := mapper.WorkspaceToResource(w)
	return &resource, nil
}
