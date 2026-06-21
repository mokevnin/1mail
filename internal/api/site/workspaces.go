package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/workspace"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
)

func toWorkspaceResource(w *ent.Workspace) siteapi.SiteWorkspaceResource {
	return siteapi.SiteWorkspaceResource{
		ID:         siteapi.EntityId(strconv.FormatInt(w.ID, 10)),
		Name:       w.Name,
		Slug:       w.Slug,
		CollectKey: w.CollectKey,
		CreatedAt:  siteapi.Timestamp(w.CreatedAt),
	}
}

// SiteWorkspacesList returns the workspaces owned by the authenticated user.
func (h *Handlers) SiteWorkspacesList(ctx context.Context) ([]siteapi.SiteWorkspaceResource, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return []siteapi.SiteWorkspaceResource{}, nil
	}

	items, err := h.ent.Workspace.Query().
		Where(workspace.UserID(a.UserID)).
		Order(ent.Asc(workspace.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	resources := make([]siteapi.SiteWorkspaceResource, len(items))
	for i, w := range items {
		resources[i] = toWorkspaceResource(w)
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
	resource := toWorkspaceResource(w)
	return &resource, nil
}
