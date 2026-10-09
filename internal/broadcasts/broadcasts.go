// Package broadcasts owns the Broadcast send/schedule state machine (ADR 0016):
// draft -> scheduled -> sending, clearing a schedule, and reverting to draft when
// the job cannot be enqueued. Every surface that sends or schedules a Broadcast
// (the /site SPA today, /api and MCP next) calls it; none re-implements a
// transition. Handlers stay thin adapters: scope check, call, map errors.
//
// What happens after "sending" (recipients, delivery, sent/failed) is the job
// worker's business, via Outbound send (ADR 0015).
package broadcasts

import (
	"context"
	"errors"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/broadcast"
)

var (
	// ErrNotFound: no such Broadcast in this Workspace.
	ErrNotFound = errors.New("broadcasts: broadcast not found")
	// ErrNotSendable: the Broadcast is past draft/scheduled (sending, sent, failed).
	ErrNotSendable = errors.New("broadcasts: broadcast is not a draft or scheduled")
)

// Enqueuer hands a Broadcast to the job queue (river in prod, inline in tests). A nil
// scheduledAt means "as soon as possible".
type Enqueuer interface {
	EnqueueBroadcast(ctx context.Context, broadcastID int64, scheduledAt *time.Time) error
}

// Module is the Broadcast send/schedule state machine.
type Module struct {
	ent *ent.Client
	q   Enqueuer
}

func New(client *ent.Client, q Enqueuer) *Module { return &Module{ent: client, q: q} }

// Send moves a draft or scheduled Broadcast to sending (dropping any schedule) and
// enqueues it for immediate dispatch. If the enqueue fails the Broadcast reverts to
// draft, so it is never stranded in sending with no job behind it.
func (m *Module) Send(ctx context.Context, workspaceID, id int64) (*ent.Broadcast, error) {
	b, err := m.claim(ctx, workspaceID, id, func(u *ent.BroadcastUpdate) *ent.BroadcastUpdate {
		return u.SetStatus(broadcast.StatusSending).ClearScheduledAt()
	})
	if err != nil {
		return nil, err
	}
	// Status is written BEFORE the enqueue and never after: the inline adapter runs
	// the send synchronously and advances the row to "sent"; a later write would
	// clobber it.
	if err := m.q.EnqueueBroadcast(ctx, b.ID, nil); err != nil {
		m.revert(ctx, b.ID)
		return nil, err
	}
	return b, nil
}

// Schedule moves a draft or scheduled Broadcast to scheduled at `when` and enqueues
// a delayed job. If the enqueue fails the Broadcast reverts to draft with no schedule.
func (m *Module) Schedule(ctx context.Context, workspaceID, id int64, when time.Time) (*ent.Broadcast, error) {
	b, err := m.claim(ctx, workspaceID, id, func(u *ent.BroadcastUpdate) *ent.BroadcastUpdate {
		return u.SetStatus(broadcast.StatusScheduled).SetScheduledAt(when)
	})
	if err != nil {
		return nil, err
	}
	if err := m.q.EnqueueBroadcast(ctx, b.ID, &when); err != nil {
		m.revert(ctx, b.ID)
		return nil, err
	}
	return b, nil
}

// claim applies a transition atomically: the UPDATE only matches a draft or
// scheduled row in this Workspace, so two racing requests cannot both win.
func (m *Module) claim(ctx context.Context, workspaceID, id int64, set func(*ent.BroadcastUpdate) *ent.BroadcastUpdate) (*ent.Broadcast, error) {
	n, err := set(m.ent.Broadcast.Update().Where(
		broadcast.ID(id), broadcast.WorkspaceID(workspaceID),
		broadcast.StatusIn(broadcast.StatusDraft, broadcast.StatusScheduled),
	)).Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		exists, err := m.ent.Broadcast.Query().Where(broadcast.ID(id), broadcast.WorkspaceID(workspaceID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, ErrNotFound
		}
		return nil, ErrNotSendable
	}
	return m.ent.Broadcast.Get(ctx, id)
}

// revert is best effort: the caller is already returning the enqueue error.
func (m *Module) revert(ctx context.Context, id int64) {
	_ = m.ent.Broadcast.UpdateOneID(id).SetStatus(broadcast.StatusDraft).ClearScheduledAt().Exec(ctx)
}
