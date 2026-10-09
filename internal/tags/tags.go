// Package tags is the one place a Contact's Tags are listed, applied and removed
// (ADR 0016). A Tag is a presence-only, workspace-scoped label (GLOSSARY); it is
// auto-created the first time it is applied, like a Custom field. Handlers on /site
// and /api stay thin adapters: scope check, call, map. Tag-based targeting is not
// here — it is the has-Tag rule of the segments engine.
package tags

import (
	"context"
	"errors"
	"strings"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/contact"
	"github.com/mokevnin/1mail/ent/tag"
)

// Domain errors. Callers match with errors.Is.
var (
	// ErrContactNotFound: no such Contact in this Workspace.
	ErrContactNotFound = errors.New("tags: contact not found")
	// ErrInvalidName: the Tag name is blank.
	ErrInvalidName = errors.New("tags: name is required")
)

// Module is the tags module.
type Module struct {
	db *ent.Client
}

// New builds the tags module over an ent client.
func New(db *ent.Client) *Module { return &Module{db: db} }

// List returns the Workspace's Tag catalogue ordered by name.
func (m *Module) List(ctx context.Context, workspaceID int64) ([]*ent.Tag, error) {
	return m.db.Tag.Query().
		Where(tag.WorkspaceID(workspaceID)).
		Order(ent.Asc(tag.FieldName)).
		All(ctx)
}

// ForContact returns the Tags a Contact has, ordered by name.
func (m *Module) ForContact(ctx context.Context, workspaceID, contactID int64) ([]*ent.Tag, error) {
	if err := requireContact(ctx, m.db, workspaceID, contactID); err != nil {
		return nil, err
	}
	return m.db.Tag.Query().
		Where(tag.WorkspaceID(workspaceID), tag.HasContactsWith(contact.ID(contactID))).
		Order(ent.Asc(tag.FieldName)).
		All(ctx)
}

// Apply gives the Contact the named Tag, creating the Tag on first use. It is
// idempotent: applying a Tag the Contact already has changes nothing.
func (m *Module) Apply(ctx context.Context, workspaceID, contactID int64, name string) (*ent.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrInvalidName
	}
	tx, err := m.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	t, err := apply(ctx, tx, workspaceID, contactID, name)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	return t, tx.Commit()
}

func apply(ctx context.Context, tx *ent.Tx, workspaceID, contactID int64, name string) (*ent.Tag, error) {
	if err := requireContact(ctx, tx.Client(), workspaceID, contactID); err != nil {
		return nil, err
	}
	if err := tx.Tag.Create().
		SetWorkspaceID(workspaceID).
		SetName(name).
		OnConflictColumns(tag.FieldName, tag.FieldWorkspaceID).
		Ignore().
		Exec(ctx); err != nil {
		return nil, err
	}
	t, err := tx.Tag.Query().Where(tag.WorkspaceID(workspaceID), tag.Name(name)).Only(ctx)
	if err != nil {
		return nil, err
	}
	if err := tx.Contact.UpdateOneID(contactID).AddTagIDs(t.ID).Exec(ctx); err != nil {
		return nil, err
	}
	return t, nil
}

// Remove takes the named Tag off the Contact. It is idempotent: a Tag the Contact
// does not have (or that does not exist) is a no-op. The Tag stays in the catalogue.
func (m *Module) Remove(ctx context.Context, workspaceID, contactID int64, name string) error {
	if err := requireContact(ctx, m.db, workspaceID, contactID); err != nil {
		return err
	}
	t, err := m.db.Tag.Query().
		Where(tag.WorkspaceID(workspaceID), tag.Name(strings.TrimSpace(name))).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return m.db.Contact.UpdateOneID(contactID).RemoveTagIDs(t.ID).Exec(ctx)
}

func requireContact(ctx context.Context, db *ent.Client, workspaceID, contactID int64) error {
	ok, err := db.Contact.Query().
		Where(contact.ID(contactID), contact.WorkspaceID(workspaceID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if !ok {
		return ErrContactNotFound
	}
	return nil
}
