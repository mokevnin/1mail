package operator

import (
	"context"

	"github.com/mokevnin/sphericon/ent"
	entworkspace "github.com/mokevnin/sphericon/ent/workspace"
	"github.com/mokevnin/sphericon/internal/pagination"
)

// ListWorkspaces is one page of every Workspace, newest first, optionally only those
// whose slug contains slug (case-insensitive). It reads the Workspace rows themselves
// through the raw client: the Operator console shows tenant metadata only, never a
// Workspace's Contacts, content or Events, so no Workspace-scoped query is made.
func (m *Module) ListWorkspaces(ctx context.Context, slug string, p pagination.Params) (pagination.Page[*ent.Workspace], error) {
	if err := m.licensed(); err != nil {
		return pagination.Page[*ent.Workspace]{}, err
	}
	query := func() *ent.WorkspaceQuery {
		q := m.ent.Workspace.Query()
		if slug != "" {
			q = q.Where(entworkspace.SlugContainsFold(slug))
		}
		return q
	}
	return pagination.List(ctx, p,
		func(ctx context.Context) (int, error) { return query().Count(ctx) },
		func(ctx context.Context, limit, offset int) ([]*ent.Workspace, error) {
			return query().Order(ent.Desc(entworkspace.FieldID)).Limit(limit).Offset(offset).All(ctx)
		})
}

// GetWorkspace is one Workspace's metadata row; an ent not-found error when it does
// not exist.
func (m *Module) GetWorkspace(ctx context.Context, id int64) (*ent.Workspace, error) {
	if err := m.licensed(); err != nil {
		return nil, err
	}
	return m.ent.Workspace.Get(ctx, id)
}
