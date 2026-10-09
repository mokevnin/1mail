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
	"github.com/mokevnin/1mail/ent/predicate"
	"github.com/mokevnin/1mail/ent/segment"
)

var (
	// ErrNotFound: no such Broadcast in this Workspace.
	ErrNotFound = errors.New("broadcasts: broadcast not found")
	// ErrNotSendable: the Broadcast is past draft/scheduled (sending, sent, failed).
	ErrNotSendable = errors.New("broadcasts: broadcast is not a draft or scheduled")
	// ErrNotScheduled: the Broadcast is not scheduled, so there is no schedule to clear.
	ErrNotScheduled = errors.New("broadcasts: broadcast is not scheduled")
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

// Unschedule returns a scheduled Broadcast to draft and drops its schedule. The
// delayed job stays queued; when it fires it finds a draft and does nothing.
func (m *Module) Unschedule(ctx context.Context, workspaceID, id int64) (*ent.Broadcast, error) {
	n, err := m.ent.Broadcast.Update().
		Where(broadcast.ID(id), broadcast.WorkspaceID(workspaceID), broadcast.StatusEQ(broadcast.StatusScheduled)).
		SetStatus(broadcast.StatusDraft).ClearScheduledAt().Save(ctx)
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
		return nil, ErrNotScheduled
	}
	return m.ent.Broadcast.Get(ctx, id)
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

// --- Authoring (ADR 0016): drafts, audience, report ---

var (
	// ErrNotDraft: the Broadcast is past draft (scheduled, sending, sent, failed) and
	// can no longer be edited, re-targeted or deleted through the authoring surface.
	ErrNotDraft = errors.New("broadcasts: broadcast is not a draft")
	// ErrSegmentNotFound: the audience Segment does not exist in this Workspace.
	ErrSegmentNotFound = errors.New("broadcasts: segment not found")
)

// Fields are the author-editable content of a Broadcast. A nil field is "not
// provided": Create falls back to the schema default, Update keeps the stored value.
type Fields struct {
	Name      *string
	Subject   *string
	FromName  *string
	FromEmail *string
	Body      *string
}

// Create makes a draft Broadcast. Nothing here schedules or sends.
func (m *Module) Create(ctx context.Context, workspaceID int64, f Fields) (*ent.Broadcast, error) {
	q := m.ent.Broadcast.Create().SetWorkspaceID(workspaceID).
		SetNillableFromName(f.FromName).
		SetNillableFromEmail(f.FromEmail)
	if f.Name != nil {
		q.SetName(*f.Name)
	}
	if f.Subject != nil {
		q.SetSubject(*f.Subject)
	}
	if f.Body != nil {
		q.SetBody(*f.Body)
	}
	return q.Save(ctx)
}

// Get returns one Broadcast of the Workspace.
func (m *Module) Get(ctx context.Context, workspaceID, id int64) (*ent.Broadcast, error) {
	b, err := m.ent.Broadcast.Query().Where(broadcast.ID(id), broadcast.WorkspaceID(workspaceID)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return b, err
}

// List returns one page of the Workspace's Broadcasts, newest first, and the total.
func (m *Module) List(ctx context.Context, workspaceID int64, limit, offset int) ([]*ent.Broadcast, int, error) {
	q := m.ent.Broadcast.Query().Where(broadcast.WorkspaceID(workspaceID))
	total, err := q.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := q.Order(ent.Desc(broadcast.FieldID)).Limit(limit).Offset(offset).All(ctx)
	return items, total, err
}

// Update edits a draft. The UPDATE only matches a draft row, so a Broadcast that was
// claimed for sending in the meantime is never edited.
func (m *Module) Update(ctx context.Context, workspaceID, id int64, f Fields) (*ent.Broadcast, error) {
	u := m.ent.Broadcast.Update().Where(draftOf(workspaceID, id)).
		SetNillableName(f.Name).
		SetNillableSubject(f.Subject).
		SetNillableFromName(f.FromName).
		SetNillableFromEmail(f.FromEmail).
		SetNillableBody(f.Body)
	return m.editDraft(ctx, workspaceID, id, u)
}

// SetAudience points a draft at a Segment, or at all active contacts when segmentID
// is nil. It never schedules or sends.
func (m *Module) SetAudience(ctx context.Context, workspaceID, id int64, segmentID *int64) (*ent.Broadcast, error) {
	if segmentID != nil {
		ok, err := m.ent.Segment.Query().Where(segment.ID(*segmentID), segment.WorkspaceID(workspaceID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !ok {
			// A foreign or unknown broadcast is still reported as such first.
			if _, err := m.Get(ctx, workspaceID, id); err != nil {
				return nil, err
			}
			return nil, ErrSegmentNotFound
		}
	}
	u := m.ent.Broadcast.Update().Where(draftOf(workspaceID, id)).SetNillableSegmentID(segmentID)
	if segmentID == nil {
		u.ClearSegmentID()
	}
	return m.editDraft(ctx, workspaceID, id, u)
}

// DeleteDraft removes a draft Broadcast. Past draft the Broadcast is history (its
// recipients and report) and is not removable here.
func (m *Module) DeleteDraft(ctx context.Context, workspaceID, id int64) error {
	n, err := m.ent.Broadcast.Delete().Where(draftOf(workspaceID, id)).Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return m.notDraftOrNotFound(ctx, workspaceID, id)
	}
	return nil
}

// Report is a Broadcast's delivery report. Rates are ratios in [0,1]; a zero
// denominator yields 0.
type Report struct {
	Status       broadcast.Status
	HoldReason   *string
	Recipients   int
	Sent         int
	Skipped      int
	Failed       int
	Opened       int
	Clicked      int
	Unsubscribed int
	OpenRate     float32
	ClickRate    float32
}

// Report reads the Broadcast's denormalized delivery counters.
func (m *Module) Report(ctx context.Context, workspaceID, id int64) (Report, error) {
	b, err := m.Get(ctx, workspaceID, id)
	if err != nil {
		return Report{}, err
	}
	return Report{
		Status: b.Status, HoldReason: b.HoldReason,
		Recipients: b.RecipientsTotal, Sent: b.SentCount, Skipped: b.SkippedCount, Failed: b.FailedCount,
		Opened: b.OpenedCount, Clicked: b.ClickedCount, Unsubscribed: b.UnsubscribedCount,
		OpenRate: ratio(b.OpenedCount, b.SentCount), ClickRate: ratio(b.ClickedCount, b.SentCount),
	}, nil
}

func ratio(num, denom int) float32 {
	if denom <= 0 {
		return 0
	}
	return float32(num) / float32(denom)
}

func draftOf(workspaceID, id int64) predicate.Broadcast {
	return broadcast.And(broadcast.ID(id), broadcast.WorkspaceID(workspaceID), broadcast.StatusEQ(broadcast.StatusDraft))
}

func (m *Module) editDraft(ctx context.Context, workspaceID, id int64, u *ent.BroadcastUpdate) (*ent.Broadcast, error) {
	n, err := u.Save(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		return nil, m.notDraftOrNotFound(ctx, workspaceID, id)
	}
	return m.Get(ctx, workspaceID, id)
}

func (m *Module) notDraftOrNotFound(ctx context.Context, workspaceID, id int64) error {
	if _, err := m.Get(ctx, workspaceID, id); err != nil {
		return err
	}
	return ErrNotDraft
}
