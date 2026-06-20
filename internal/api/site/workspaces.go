package site

import (
	"context"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/workspace"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
)

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
		resources[i] = siteapi.SiteWorkspaceResource{
			ID:         siteapi.EntityId(strconv.FormatInt(w.ID, 10)),
			Name:       w.Name,
			Slug:       w.Slug,
			CollectKey: w.CollectKey,
			CreatedAt:  siteapi.Timestamp(w.CreatedAt),
		}
	}
	return resources, nil
}
