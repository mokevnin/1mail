package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/workspace"
)

// SuspendWorkspace freezes all outbound sending for a Workspace (ADR 0007): it sets
// suspended_at together with who did it (an Operator id, "cli", or "system" for the
// automated abuse detector) and why. The send path reads it as a reversible Hold
// (ADR 0015); login, reads and /collect are unaffected. It reports whether anything
// changed: suspending an already-suspended Workspace is a no-op, so the first actor
// and reason — the ones the owner was told about — are never overwritten.
func SuspendWorkspace(ctx context.Context, client *ent.Client, workspaceID int64, by, reason string) (bool, error) {
	by, reason = strings.TrimSpace(by), strings.TrimSpace(reason)
	if by == "" || reason == "" {
		return false, errors.New("suspend workspace: an actor and a reason are required")
	}
	n, err := client.Workspace.Update().
		Where(workspace.ID(workspaceID), workspace.SuspendedAtIsNil()).
		SetSuspendedAt(time.Now()).
		SetSuspendedBy(by).
		SetSuspensionReason(reason).
		Save(ctx)
	if err != nil {
		return false, fmt.Errorf("suspend workspace %d: %w", workspaceID, err)
	}
	return n > 0, nil
}

// UnsuspendWorkspace lifts a suspension, clearing its attribution. It reports whether
// the Workspace was suspended. Held sends resume on their own: the jobs that were
// snoozed ask Outbound send again.
func UnsuspendWorkspace(ctx context.Context, client *ent.Client, workspaceID int64) (bool, error) {
	n, err := client.Workspace.Update().
		Where(workspace.ID(workspaceID), workspace.SuspendedAtNotNil()).
		ClearSuspendedAt().
		ClearSuspendedBy().
		ClearSuspensionReason().
		Save(ctx)
	if err != nil {
		return false, fmt.Errorf("unsuspend workspace %d: %w", workspaceID, err)
	}
	return n > 0, nil
}

// WorkspaceIDBySlug resolves a Workspace slug to its id (operator tooling addresses
// Workspaces by slug).
func WorkspaceIDBySlug(ctx context.Context, client *ent.Client, slug string) (int64, error) {
	id, err := client.Workspace.Query().Where(workspace.Slug(slug)).OnlyID(ctx)
	if err != nil {
		return 0, fmt.Errorf("workspace %q: %w", slug, err)
	}
	return id, nil
}
