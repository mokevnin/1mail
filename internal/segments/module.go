package segments

import (
	"context"
	"errors"
	"fmt"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/segment"
)

// Domain errors returned by Module. Callers match with errors.Is.
var (
	// ErrInvalidDefinition: the rule definition does not parse or does not fit the
	// Contact schema. The wrapped message is safe to show to the author.
	ErrInvalidDefinition = errors.New("segments: invalid definition")
	// ErrNotFound: no such Segment in this Workspace.
	ErrNotFound = errors.New("segments: segment not found")
)

// Module is the one place a Segment is written, validated, previewed and counted
// (ADR 0016). Create and update share one validation rule set; handlers stay thin
// adapters (scope check, call, map). Membership is the rule alone — Send-eligibility
// is subtracted only at send, never folded into a count (ADR 0001).
type Module struct {
	db *ent.Client
}

// New builds the Segments module over an ent client.
func New(db *ent.Client) *Module { return &Module{db: db} }

// CreateInput is a new Segment. A nil or empty Definition means "no rules yet".
type CreateInput struct {
	Name       string
	Type       segment.Type
	Definition *string
}

// UpdateInput changes a Segment; nil fields are left as they are.
type UpdateInput struct {
	Name       *string
	Type       *segment.Type
	Definition *string
}

// Validate checks a definition against the Contact schema. An empty definition is
// valid. This is the single rule set behind Create, Update, Preview and Count.
func (s *Module) Validate(def string) error {
	if def == "" {
		return nil
	}
	if err := ValidateContactDefinition(def); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidDefinition, err.Error())
	}
	return nil
}

// Create validates the definition (whatever the Segment type) and stores the Segment.
func (s *Module) Create(ctx context.Context, workspaceID int64, in CreateInput) (*ent.Segment, error) {
	if in.Definition != nil {
		if err := s.Validate(*in.Definition); err != nil {
			return nil, err
		}
	}
	return s.db.Segment.Create().
		SetWorkspaceID(workspaceID).
		SetName(in.Name).
		SetType(in.Type).
		SetNillableDefinition(in.Definition).
		Save(ctx)
}

// Update validates a changed definition exactly as Create does, then applies the change.
func (s *Module) Update(ctx context.Context, workspaceID, id int64, in UpdateInput) (*ent.Segment, error) {
	if in.Definition != nil {
		if err := s.Validate(*in.Definition); err != nil {
			return nil, err
		}
	}
	q := s.db.Segment.UpdateOneID(id).
		Where(segment.WorkspaceID(workspaceID)).
		SetNillableName(in.Name).
		SetNillableDefinition(in.Definition)
	if in.Type != nil {
		q = q.SetType(*in.Type)
	}
	seg, err := q.Save(ctx)
	if ent.IsNotFound(err) {
		return nil, ErrNotFound
	}
	return seg, err
}

// Preview counts the Contacts in the Workspace that match an unsaved definition.
func (s *Module) Preview(ctx context.Context, workspaceID int64, def string) (int, error) {
	if err := s.Validate(def); err != nil {
		return 0, err
	}
	pred, err := ContactPredicate(def)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", ErrInvalidDefinition, err.Error())
	}
	return s.db.Contact.Query().Where(contact.WorkspaceID(workspaceID), pred).Count(ctx)
}

// Count counts the Contacts matching a stored Segment, evaluated live.
func (s *Module) Count(ctx context.Context, workspaceID, id int64) (int, error) {
	seg, err := s.db.Segment.Query().
		Where(segment.ID(id), segment.WorkspaceID(workspaceID)).
		Only(ctx)
	if ent.IsNotFound(err) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, err
	}
	def := ""
	if seg.Definition != nil {
		def = *seg.Definition
	}
	return s.Preview(ctx, workspaceID, def)
}
