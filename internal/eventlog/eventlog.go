// Package eventlog is the one place a customer's Events enter and are summarized
// (ADR 0016, use-case modules). Ingest resolves each Event's identity to an existing
// Contact (ADR 0002) and publishes the batch in a single transaction; Actions is the
// workspace's distinct-action query. The /site and /api handlers are thin adapters
// over it and no longer duplicate either piece.
//
// The persist subscriber writes the Event rows asynchronously from the outbox, so
// Ingest is accept-then-process.
package eventlog

import (
	"context"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/service"
)

// Module is the Events ingest and read module.
type Module struct {
	client *ent.Client
	bus    *events.Bus
}

// New builds the module over the ent client (reads) and the domain-event bus
// (transactional publish).
func New(client *ent.Client, bus *events.Bus) *Module {
	return &Module{client: client, bus: bus}
}

// Input is one customer Event to ingest. Action and Properties are the customer's
// own and are stored as-is; the alias keys (SubjectID, Email, Phone) identify the
// Contact it belongs to, if one exists.
type Input struct {
	SubjectID  string
	Email      *string
	Phone      *string
	Action     string
	OccurredAt time.Time // zero => publish time
	Properties map[string]any
}

// Ingest attaches each Event to an existing Contact by stable identity (an unknown
// identity stays anonymous; a Contact is never created) and publishes the whole batch
// atomically: either every Event is accepted or none is.
func (m *Module) Ingest(ctx context.Context, workspaceID int64, inputs []Input) error {
	return m.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		for _, in := range inputs {
			contactID, err := service.ResolveContactID(ctx, tx, workspaceID, in.SubjectID, in.Email, in.Phone)
			if err != nil {
				return err
			}
			collected := &events.CollectedEvent{
				WorkspaceID: workspaceID,
				ContactID:   contactID,
				SubjectID:   in.SubjectID,
				Action:      in.Action,
				OccurredAt:  in.OccurredAt,
				Properties:  in.Properties,
			}
			if in.Email != nil {
				collected.Email = *in.Email
			}
			if in.Phone != nil {
				collected.Phone = *in.Phone
			}
			if err := pub.Publish(ctx, collected); err != nil {
				return err
			}
		}
		return nil
	})
}

// Actions returns the workspace's distinct Event actions, sorted.
func (m *Module) Actions(ctx context.Context, workspaceID int64) ([]string, error) {
	return m.client.Event.Query().
		Where(event.WorkspaceID(workspaceID)).
		Order(ent.Asc(event.FieldAction)).
		GroupBy(event.FieldAction).
		Strings(ctx)
}
