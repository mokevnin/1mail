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
	"errors"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/event"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/service"
)

// Module is the Events ingest and read module.
type Module struct {
	bus *events.Bus
}

// New builds the module over the domain-event bus (transactional publish). Reads
// and identity resolution go through the Workspace-scoped client each call receives.
func New(bus *events.Bus) *Module {
	return &Module{bus: bus}
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
func (m *Module) Ingest(ctx context.Context, s *ent.Scoped, inputs []Input) error {
	return m.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, pub events.Publisher) error {
		for _, in := range inputs {
			contactID, err := service.ResolveContactID(ctx, ts, in.SubjectID, in.Email, in.Phone)
			if err != nil {
				return err
			}
			collected := &events.CollectedEvent{
				WorkspaceID: ts.WorkspaceID(),
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

// ErrInvalid means an Event lacks its subject id or its action.
var ErrInvalid = errors.New("eventlog: subject id and action are required")

// IngestEach ingests each Event in its own transaction, so an invalid or failing item
// does not affect the others (Ingest, by contrast, is all-or-nothing). The returned
// errors are parallel to inputs; nil means the Event was accepted.
func (m *Module) IngestEach(ctx context.Context, s *ent.Scoped, inputs []Input) []error {
	errs := make([]error, len(inputs))
	for i, in := range inputs {
		if strings.TrimSpace(in.SubjectID) == "" || strings.TrimSpace(in.Action) == "" {
			errs[i] = ErrInvalid
			continue
		}
		errs[i] = m.Ingest(ctx, s, []Input{in})
	}
	return errs
}

// Actions returns the workspace's distinct Event actions, sorted.
func (m *Module) Actions(ctx context.Context, s *ent.Scoped) ([]string, error) {
	return s.Event().Query().
		Order(ent.Asc(event.FieldAction)).
		GroupBy(event.FieldAction).
		Strings(ctx)
}
