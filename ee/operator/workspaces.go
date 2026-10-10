package operator

import (
	"context"
	"errors"
	"log/slog"
	"strconv"
	"strings"

	"github.com/mokevnin/sphericon/ent"
	entworkspace "github.com/mokevnin/sphericon/ent/workspace"
	"github.com/mokevnin/sphericon/internal/events"
	"github.com/mokevnin/sphericon/internal/jobs"
	"github.com/mokevnin/sphericon/internal/messaging"
	"github.com/mokevnin/sphericon/internal/pagination"
	"github.com/mokevnin/sphericon/internal/suspension"
)

// ErrReasonRequired refuses a suspension with no reason: the owner is told why.
var ErrReasonRequired = errors.New("a suspension needs a reason")

// SuspendWorkspace freezes the Workspace's outbound sending as the Operator, through
// the same core mechanism the command line uses (ADR 0007), then tells its owners via
// the system sender. It reports whether anything changed: an already-suspended
// Workspace is left as it was and its owners are not told twice. A failed notice does
// not undo the suspension; it is logged. A blank reason is ErrReasonRequired.
func (m *Module) SuspendWorkspace(ctx context.Context, bus *events.Bus, sender messaging.EmailSender, id int64, by *ent.Operator, reason string) (*ent.Workspace, bool, error) {
	if err := m.licensed(); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(reason) == "" {
		return nil, false, ErrReasonRequired
	}
	if _, err := m.ent.Workspace.Get(ctx, id); err != nil {
		return nil, false, err
	}
	changed, err := suspension.SuspendWorkspace(ctx, bus, id, suspension.Operator(strconv.FormatInt(by.ID, 10)), reason)
	if err != nil {
		return nil, false, err
	}
	if changed {
		if err := jobs.NotifyWorkspaceSuspended(ctx, m.ent, sender, id); err != nil {
			slog.ErrorContext(ctx, "workspace suspended, but the owner notice could not be sent", "workspace_id", id, "error", err)
		}
	}
	ws, err := m.ent.Workspace.Get(ctx, id)
	return ws, changed, err
}

// UnsuspendWorkspace lifts the Workspace's suspension as the Operator; held sends
// resume on their own. It reports whether the Workspace was suspended.
func (m *Module) UnsuspendWorkspace(ctx context.Context, bus *events.Bus, id int64, by *ent.Operator) (*ent.Workspace, bool, error) {
	if err := m.licensed(); err != nil {
		return nil, false, err
	}
	if _, err := m.ent.Workspace.Get(ctx, id); err != nil {
		return nil, false, err
	}
	changed, err := suspension.UnsuspendWorkspace(ctx, bus, id, suspension.Operator(strconv.FormatInt(by.ID, 10)))
	if err != nil {
		return nil, false, err
	}
	ws, err := m.ent.Workspace.Get(ctx, id)
	return ws, changed, err
}

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
