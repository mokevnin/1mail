// Package erasure is the one place a Contact's personal data is erased (GDPR Art. 17,
// ADR 0021). Deleting a Contact is Erasure: there is no softer variant. Every surface
// (the /site SPA API, the external /api and so MCP) hands it a Workspace-scoped client
// and an Identifier and gets back nil or a domain error; none re-implements what is
// removed, anonymized or kept.
//
// The whole erasure is one transaction, opened through events.Bus.WithinScopedTx so
// the Contact change and the domain-event outbox commit together. Inside it the work
// is an ordered list of steps (see Module.steps), each a small function over the
// resolved Target; extending Erasure (another identifier kind, another dependent
// table, another signal published in the same transaction) means resolving the new
// identifier into a Target or adding one step, not a second code path.
package erasure

import (
	"context"
	"errors"
	"strconv"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/internal/events"
)

// ErrNotFound means the identifier resolves to nothing in the Workspace.
var ErrNotFound = errors.New("erasure: not found")

// Identifier names the data subject to erase. Build one with ByContactID.
type Identifier struct {
	contactID int64
}

// ByContactID identifies a Contact by its id.
func ByContactID(id int64) Identifier { return Identifier{contactID: id} }

// kind names the identifier for the contact.erased Event.
func (Identifier) kind() string { return "contact_id" }

// Operator is who asked for the Erasure: a User (Site) or an API token (External, MCP).
type Operator struct {
	// Kind is OperatorUser or OperatorAPIToken.
	Kind string
	ID   int64
}

// Operator kinds, as recorded on the contact.erased Event.
const (
	OperatorUser     = "user"
	OperatorAPIToken = "api_token"
)

// Target is a resolved data subject: what every step erases. Resolving an Identifier
// into a Target is the only place the identifier kind matters.
type Target struct {
	// ContactID is the Contact being erased.
	ContactID int64
	// VisitorIDs are the anonymous devices bound to the Contact; their Events are
	// erased with it.
	VisitorIDs []string
	// Destinations are the Contact's addresses.
	Destinations []string
	// SubjectID is the customer's own id for the person (the Contact's subject_id, or
	// its 1mail id when it has none); only the contact.erased delivery carries it.
	SubjectID string
	// IdentifierKind is what the Identifier was.
	IdentifierKind string
	// Operator is who asked.
	Operator Operator
}

// step is one part of the erasure, run inside the transaction.
type step func(ctx context.Context, s *ent.Scoped, t *Target, pub events.Publisher) error

// Module erases data subjects.
type Module struct {
	bus *events.Bus
}

// New builds the module over the bus whose transaction every erasure runs in.
func New(bus *events.Bus) *Module {
	return &Module{bus: bus}
}

// Erase erases the identified data subject in one transaction: either all of it
// happens or none. ErrNotFound means the identifier resolves to nothing in the
// Workspace (including a row of another Workspace).
func (m *Module) Erase(ctx context.Context, s *ent.Scoped, id Identifier, op Operator) error {
	return m.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, pub events.Publisher) error {
		target, err := resolve(ctx, ts, id)
		if err != nil {
			return err
		}
		target.Operator = op
		for _, run := range m.steps() {
			if err := run(ctx, ts, target, pub); err != nil {
				return err
			}
		}
		return nil
	})
}

// steps is the ordered erasure. The Contact row goes last: the other steps find the
// Contact's rows through ids already resolved into the Target.
func (m *Module) steps() []step {
	return []step{
		eraseEvents,
		eraseVisitors,
		eraseConfirmations,
		eraseRuns,
		detachOptOuts,
		anonymizeDelivery,
		eraseContact,
		publishErased,
	}
}

func resolve(ctx context.Context, s *ent.Scoped, id Identifier) (*Target, error) {
	c, err := s.Contact().Get(ctx, id.contactID)
	if ent.IsNotFound(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	visitors, err := s.Visitor().Query().Where(visitorOf(c.ID)).All(ctx)
	if err != nil {
		return nil, err
	}
	t := &Target{ContactID: c.ID, IdentifierKind: id.kind(), SubjectID: strconv.FormatInt(c.ID, 10)}
	if c.SubjectID != nil && *c.SubjectID != "" {
		t.SubjectID = *c.SubjectID
	}
	for _, v := range visitors {
		t.VisitorIDs = append(t.VisitorIDs, v.VisitorID)
	}
	if c.Email != nil {
		t.Destinations = append(t.Destinations, *c.Email)
	}
	return t, nil
}
