// Package accounts is the raw-client home for what the scoped client cannot
// express (ADR 0017): actors whose Workspace is not known up front. It owns the
// User (identity, credentials), the Membership lookup that turns a /w/{slug} into
// a Workspace scope, the Workspace itself (it is the tenant, not a tenant-owned
// entity, so the scoped client has no wrapper for it), and the pending-Invitation
// lookup by token (the token is the only handle). The site handlers use it
// instead of holding a raw *ent.Client.
package accounts

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/invitation"
	"github.com/mokevnin/1mail/ent/membership"
	"github.com/mokevnin/1mail/ent/user"
	"github.com/mokevnin/1mail/ent/workspace"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/service"
)

// Accounts reads and writes Users, Workspaces and Memberships over the raw client.
type Accounts struct {
	ent *ent.Client
	bus *events.Bus
}

// New builds Accounts over the ent client; the bus opens the transactions that
// span several writes.
func New(client *ent.Client, bus *events.Bus) *Accounts {
	return &Accounts{ent: client, bus: bus}
}

// Scope resolves the Workspace addressed by a /w/{slug} path segment for a User and
// returns its scoped client with the User's Role. It is the site's construction
// point of the scoped client: access is "does this User have a Membership on this
// Workspace?". An ent NotFound error means the slug does not exist or the User is
// not a member.
func (a *Accounts) Scope(ctx context.Context, userID int64, slug string) (*ent.Scoped, membership.Role, error) {
	m, err := a.ent.Membership.Query().
		Where(membership.UserID(userID), membership.HasWorkspaceWith(workspace.Slug(slug))).
		Only(ctx)
	if err != nil {
		return nil, "", err
	}
	return a.ent.Scoped(m.WorkspaceID), m.Role, nil
}

// BootstrapScope is the scoped client of the oldest Workspace, for the bootstrap
// token (no caller token exists yet to say which Workspace is meant).
func (a *Accounts) BootstrapScope(ctx context.Context) (*ent.Scoped, error) {
	id, err := a.ent.Workspace.Query().Order(ent.Asc(workspace.FieldID)).FirstID(ctx)
	if err != nil {
		return nil, err
	}
	return a.ent.Scoped(id), nil
}

// WorkspacesOf lists the Workspaces a User is a member of, oldest first.
func (a *Accounts) WorkspacesOf(ctx context.Context, userID int64) ([]*ent.Workspace, error) {
	return a.ent.Workspace.Query().
		Where(workspace.HasMembershipsWith(membership.UserID(userID))).
		Order(ent.Asc(workspace.FieldID)).
		All(ctx)
}

// UpdateWorkspace renames the scoped Workspace and, when postalAddress is non-nil,
// sets (or clears, with "") its postal address.
func (a *Accounts) UpdateWorkspace(ctx context.Context, s *ent.Scoped, name string, postalAddress *string) (*ent.Workspace, error) {
	upd := a.ent.Workspace.UpdateOneID(s.WorkspaceID()).SetName(name)
	if postalAddress != nil {
		upd = upd.SetPostalAddress(*postalAddress)
	}
	return upd.Save(ctx)
}

// ChangeMembershipRole sets a Membership's Role and records `membership.update` with
// the role change as an Audit entry in the same transaction (ADR 0022): a rolled-back
// change leaves no entry, a committed one cannot lose it. name is the member's display
// name, snapshotted into the entry.
func (a *Accounts) ChangeMembershipRole(ctx context.Context, s *ent.Scoped, actor events.Actor, target *ent.Membership, name string, desired membership.Role) (*ent.Membership, error) {
	var updated *ent.Membership
	err := a.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, pub events.Publisher) error {
		m, err := ts.Membership().UpdateOneID(target.ID).SetRole(desired).Save(ctx)
		if err != nil {
			return err
		}
		updated = m
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: s.WorkspaceID(),
			Actor:       actor,
			Action:      "membership.update",
			TargetType:  "membership",
			TargetID:    strconv.FormatInt(target.ID, 10),
			TargetName:  name,
			Diff:        map[string]any{"role": map[string]any{"from": string(target.Role), "to": string(desired)}},
		})
	})
	return updated, err
}

// CreateWorkspace creates a User's initial Workspace with a unique slug derived from
// name (falling back to "workspace"). Slugs are globally unique, so on collision a
// numeric suffix is appended. The Workspace and the creator's owner Membership are
// created atomically: a Workspace is reached through Memberships, so one without its
// owner row would be inaccessible.
func (a *Accounts) CreateWorkspace(ctx context.Context, userID int64, name string) (*ent.Workspace, error) {
	base := service.Slugify(name)
	if base == "" {
		base = "workspace"
	}
	for i := 0; ; i++ {
		slug := base
		if i > 0 {
			slug = fmt.Sprintf("%s-%d", base, i+1)
		}
		exists, err := a.ent.Workspace.Query().Where(workspace.Slug(slug)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			continue
		}
		collectKey, err := service.GenerateCollectKey()
		if err != nil {
			return nil, err
		}
		ingestKey, err := service.GenerateIngestKey()
		if err != nil {
			return nil, err
		}
		var ws *ent.Workspace
		err = a.bus.WithinTx(ctx, func(tx *ent.Client, _ events.Publisher) error {
			created, cerr := tx.Workspace.Create().
				SetName(name).
				SetSlug(slug).
				SetCollectKey(collectKey).
				SetIngestKey(ingestKey).
				Save(ctx)
			if cerr != nil {
				return cerr
			}
			if _, merr := tx.Membership.Create().
				SetUserID(userID).
				SetWorkspaceID(created.ID).
				SetRole(membership.RoleOwner).
				Save(ctx); merr != nil {
				return merr
			}
			ws = created
			return nil
		})
		if service.IsUniqueViolation(err) {
			continue // lost a race on the slug; try the next suffix
		}
		if err != nil {
			return nil, err
		}
		return ws, nil
	}
}

// User loads a User by id.
func (a *Accounts) User(ctx context.Context, id int64) (*ent.User, error) {
	return a.ent.User.Get(ctx, id)
}

// UserByEmail loads a User by login email (an ent NotFound error when none).
func (a *Accounts) UserByEmail(ctx context.Context, email string) (*ent.User, error) {
	return a.ent.User.Query().Where(user.Email(email)).Only(ctx)
}

// EmailTaken reports whether an account already uses the login email.
func (a *Accounts) EmailTaken(ctx context.Context, email string) (bool, error) {
	return a.ent.User.Query().Where(user.Email(email)).Exist(ctx)
}

// CreateUser registers a User with a password hash.
func (a *Accounts) CreateUser(ctx context.Context, name, email, passwordHash string) (*ent.User, error) {
	return a.ent.User.Create().SetName(name).SetEmail(email).SetPasswordHash(passwordHash).Save(ctx)
}

// UpdateProfile sets the User's name and/or password hash (nil leaves a field
// unchanged) and returns the stored User.
func (a *Accounts) UpdateProfile(ctx context.Context, id int64, name, passwordHash *string) (*ent.User, error) {
	upd := a.ent.User.UpdateOneID(id)
	if name != nil {
		upd = upd.SetName(*name)
	}
	if passwordHash != nil {
		upd = upd.SetPasswordHash(*passwordHash)
	}
	return upd.Save(ctx)
}

// SetPassword replaces the User's password hash.
func (a *Accounts) SetPassword(ctx context.Context, id int64, passwordHash string) error {
	return a.ent.User.UpdateOneID(id).SetPasswordHash(passwordHash).Exec(ctx)
}

// MarkEmailVerified stamps the User's email as verified now.
func (a *Accounts) MarkEmailVerified(ctx context.Context, id int64) error {
	return a.ent.User.UpdateOneID(id).SetEmailVerifiedAt(time.Now()).Exec(ctx)
}

// ChangeEmail swaps the login email for one proven by a confirmation link, so it is
// stored verified. The unique index guards a race on the address (see
// service.IsUniqueViolation).
func (a *Accounts) ChangeEmail(ctx context.Context, id int64, newEmail string) error {
	return a.ent.User.UpdateOneID(id).SetEmail(newEmail).SetEmailVerifiedAt(time.Now()).Exec(ctx)
}

// PendingInvitation looks up a pending, unexpired Invitation by its raw token, with
// its Workspace edge. The token is the authorization, so no Workspace is known up
// front. An ent NotFound error when there is none.
func (a *Accounts) PendingInvitation(ctx context.Context, token string) (*ent.Invitation, error) {
	return a.ent.Invitation.Query().
		Where(
			invitation.TokenHash(service.HashInviteToken(token)),
			invitation.AcceptedAtIsNil(),
			invitation.ExpiresAtGT(time.Now()),
		).
		WithWorkspace().
		Only(ctx)
}

// AcceptInvitation turns a pending Invitation into a Membership: it attaches the
// existing User of the invited email or creates one (name + password, already
// email-verified because the link proves control of the address), joins them to the
// Workspace (a duplicate is not an error) and marks the Invitation accepted.
func (a *Accounts) AcceptInvitation(ctx context.Context, inv *ent.Invitation, name, password string) error {
	role := membership.Role(inv.Role)
	return a.bus.WithinTx(ctx, func(tx *ent.Client, _ events.Publisher) error {
		u, uerr := tx.User.Query().Where(user.Email(inv.Email)).Only(ctx)
		if ent.IsNotFound(uerr) {
			hash, herr := service.HashPassword(password)
			if herr != nil {
				return herr
			}
			u, uerr = tx.User.Create().
				SetName(name).
				SetEmail(inv.Email).
				SetPasswordHash(hash).
				SetEmailVerifiedAt(time.Now()).
				Save(ctx)
		}
		if uerr != nil {
			return uerr
		}

		ts := tx.Scoped(inv.WorkspaceID)
		if _, merr := ts.Membership().Create().
			SetUserID(u.ID).
			SetRole(role).
			Save(ctx); merr != nil && !service.IsUniqueViolation(merr) {
			return merr
		}

		return ts.Invitation().UpdateOneID(inv.ID).SetAcceptedAt(time.Now()).Exec(ctx)
	})
}
