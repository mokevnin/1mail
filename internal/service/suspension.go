package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/workspace"
	"github.com/mokevnin/1mail/internal/events"
)

// SuspendWorkspace freezes all outbound sending for a Workspace (ADR 0007): it sets
// suspended_at together with who did it (an Operator id, "cli", or "system" for the
// automated abuse detector) and why. The send path reads it as a reversible Hold
// (ADR 0015); login, reads and /collect are unaffected. It reports whether anything
// changed: suspending an already-suspended Workspace is a no-op, so the first actor
// and reason — the ones the owner was told about — are never overwritten. A change is
// recorded as `workspace.suspend` in the Workspace's Audit log in the same transaction.
func SuspendWorkspace(ctx context.Context, bus *events.Bus, workspaceID int64, by, reason string) (bool, error) {
	by, reason = strings.TrimSpace(by), strings.TrimSpace(reason)
	if by == "" || reason == "" {
		return false, errors.New("suspend workspace: an actor and a reason are required")
	}
	changed := false
	err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		n, err := tx.Workspace.Update().
			Where(workspace.ID(workspaceID), workspace.SuspendedAtIsNil()).
			SetSuspendedAt(time.Now()).
			SetSuspendedBy(by).
			SetSuspensionReason(reason).
			Save(ctx)
		if err != nil || n == 0 {
			return err
		}
		changed = true
		return recordWorkspaceAudit(ctx, tx, pub, workspaceID, by, events.ActionWorkspaceSuspend,
			map[string]any{"suspended": map[string]any{"from": false, "to": true}, "reason": map[string]any{"to": reason}})
	})
	if err != nil {
		return false, fmt.Errorf("suspend workspace %d: %w", workspaceID, err)
	}
	return changed, nil
}

// UnsuspendWorkspace lifts a suspension, clearing its attribution, and records
// `workspace.unsuspend` for the Operator `by`. It reports whether the Workspace was
// suspended. Held sends resume on their own: the jobs that were snoozed ask Outbound
// send again.
func UnsuspendWorkspace(ctx context.Context, bus *events.Bus, workspaceID int64, by string) (bool, error) {
	changed := false
	err := bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		n, err := tx.Workspace.Update().
			Where(workspace.ID(workspaceID), workspace.SuspendedAtNotNil()).
			ClearSuspendedAt().
			ClearSuspendedBy().
			ClearSuspensionReason().
			Save(ctx)
		if err != nil || n == 0 {
			return err
		}
		changed = true
		return recordWorkspaceAudit(ctx, tx, pub, workspaceID, by, events.ActionWorkspaceUnsuspend,
			map[string]any{"suspended": map[string]any{"from": true, "to": false}})
	})
	if err != nil {
		return false, fmt.Errorf("unsuspend workspace %d: %w", workspaceID, err)
	}
	return changed, nil
}

// recordWorkspaceAudit publishes the Audit entry of a suspension change. "system" (the
// automated abuse detector) is the system actor; anything else is a platform Operator,
// which the customer sees only as "1mail staff" (ADR 0022).
func recordWorkspaceAudit(ctx context.Context, tx *ent.Client, pub events.Publisher, workspaceID int64, by, action string, diff map[string]any) error {
	ws, err := tx.Workspace.Get(ctx, workspaceID)
	if err != nil {
		return err
	}
	actor := events.Actor{Kind: events.ActorOperator, ID: by}
	if by == "system" {
		actor = events.Actor{Kind: events.ActorSystem}
	}
	return events.RecordAudit(ctx, pub, &events.AuditEntry{
		WorkspaceID: workspaceID,
		Actor:       actor,
		Action:      action,
		TargetType:  "workspace",
		TargetID:    strconv.FormatInt(workspaceID, 10),
		TargetName:  ws.Name,
		Diff:        diff,
	})
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
