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
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/invitation"
	"github.com/mokevnin/1mail/ent/membership"
	"github.com/mokevnin/1mail/ent/user"
	"github.com/mokevnin/1mail/ent/workspace"
	"github.com/mokevnin/1mail/internal/credentials"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/events"
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
		WithUser().
		Only(ctx)
	if err != nil {
		return nil, "", err
	}
	actor := events.Actor{Kind: events.ActorUser, ID: strconv.FormatInt(userID, 10)}
	if u := m.Edges.User; u != nil {
		actor.Name = u.Name
	}
	return a.bus.Act(a.ent.Scoped(m.WorkspaceID), actor), m.Role, nil
}

// BootstrapScope is the scoped client of the oldest Workspace, for the bootstrap
// token (no caller token exists yet to say which Workspace is meant).
func (a *Accounts) BootstrapScope(ctx context.Context) (*ent.Scoped, error) {
	id, err := a.ent.Workspace.Query().Order(ent.Asc(workspace.FieldID)).FirstID(ctx)
	if err != nil {
		return nil, err
	}
	return events.Ingest(a.ent.Scoped(id)), nil
}

// WorkspacesOf lists the Workspaces a User is a member of, oldest first.
func (a *Accounts) WorkspacesOf(ctx context.Context, userID int64) ([]*ent.Workspace, error) {
	return a.ent.Workspace.Query().
		Where(workspace.HasMembershipsWith(membership.UserID(userID))).
		Order(ent.Asc(workspace.FieldID)).
		All(ctx)
}

// UpdateWorkspace renames the scoped Workspace and, when postalAddress is non-nil,
// sets (or clears, with "") its postal address. A change is recorded as a
// `workspace.update` Audit entry in the same transaction (ADR 0022): the Workspace is
// the tenant root and has no scoped wrapper, so this is an explicit path.
func (a *Accounts) UpdateWorkspace(ctx context.Context, s *ent.Scoped, actor events.Actor, name string, postalAddress *string) (*ent.Workspace, error) {
	var updated *ent.Workspace
	err := a.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		before, err := tx.Workspace.Get(ctx, s.WorkspaceID())
		if err != nil {
			return err
		}
		upd := tx.Workspace.UpdateOneID(s.WorkspaceID()).SetName(name)
		if postalAddress != nil {
			upd = upd.SetPostalAddress(*postalAddress)
		}
		updated, err = upd.Save(ctx)
		if err != nil {
			return err
		}
		diff := map[string]any{}
		if before.Name != updated.Name {
			diff["name"] = map[string]any{"from": before.Name, "to": updated.Name}
		}
		if before.PostalAddress != updated.PostalAddress {
			diff["postal_address"] = map[string]any{"from": before.PostalAddress, "to": updated.PostalAddress}
		}
		if len(diff) == 0 {
			return nil
		}
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: s.WorkspaceID(),
			Actor:       actor,
			Action:      events.ActionWorkspaceUpdate,
			TargetType:  "workspace",
			TargetID:    strconv.FormatInt(updated.ID, 10),
			TargetName:  updated.Name,
			Diff:        diff,
		})
	})
	return updated, err
}

// SetAuditRetention sets (or, with nil, clears) the number of days the Audit log is
// kept (the EE advanced-retention window, ADR 0014) and records the change as a
// `workspace.update` Audit entry in the same transaction. The caller has checked the
// license and the role.
func (a *Accounts) SetAuditRetention(ctx context.Context, s *ent.Scoped, actor events.Actor, days *int) error {
	return a.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		before, err := tx.Workspace.Get(ctx, s.WorkspaceID())
		if err != nil {
			return err
		}
		upd := tx.Workspace.UpdateOneID(s.WorkspaceID())
		if days == nil {
			upd = upd.ClearRetentionDays()
		} else {
			upd = upd.SetRetentionDays(*days)
		}
		if _, err := upd.Save(ctx); err != nil {
			return err
		}
		if ptrEqual(before.RetentionDays, days) {
			return nil
		}
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: s.WorkspaceID(),
			Actor:       actor,
			Action:      events.ActionWorkspaceUpdate,
			TargetType:  "workspace",
			TargetID:    strconv.FormatInt(before.ID, 10),
			TargetName:  before.Name,
			Diff:        map[string]any{"retention_days": map[string]any{"from": before.RetentionDays, "to": days}},
		})
	})
}

func ptrEqual(a, b *int) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

// CreateWorkspace creates a User's initial Workspace with a unique slug derived from
// name (falling back to "workspace"). Slugs are globally unique, so on collision a
// numeric suffix is appended. The Workspace and the creator's owner Membership are
// created atomically: a Workspace is reached through Memberships, so one without its
// owner row would be inaccessible.
func (a *Accounts) CreateWorkspace(ctx context.Context, userID int64, name string) (*ent.Workspace, error) {
	base := Slugify(name)
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
		collectKey, err := credentials.GenerateCollectKey()
		if err != nil {
			return nil, err
		}
		ingestKey, err := credentials.GenerateIngestKey()
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
		if db.IsUniqueViolation(err) {
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
// unchanged) and returns the stored User. A password change is recorded as an Audit
// entry in every Workspace the User belongs to, in the same transaction.
func (a *Accounts) UpdateProfile(ctx context.Context, id int64, name, passwordHash *string) (*ent.User, error) {
	var saved *ent.User
	err := a.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		upd := tx.User.UpdateOneID(id)
		if name != nil {
			upd = upd.SetName(*name)
		}
		if passwordHash != nil {
			upd = upd.SetPasswordHash(*passwordHash)
		}
		u, err := upd.Save(ctx)
		if err != nil {
			return err
		}
		saved = u
		if passwordHash == nil {
			return nil
		}
		return recordUserAction(ctx, tx, pub, u, events.ActionUserPasswordChange, map[string]any{"password": "changed"})
	})
	return saved, err
}

// SetPassword replaces the User's password hash (a reset by emailed link) and records
// the change like UpdateProfile.
func (a *Accounts) SetPassword(ctx context.Context, id int64, passwordHash string) error {
	_, err := a.UpdateProfile(ctx, id, nil, &passwordHash)
	return err
}

// RecordLogin records a successful sign-in of the User as `user.login` in the log of
// every Workspace they belong to (ADR 0022: a login belongs to a User, not a
// Workspace, so there is no account-level log).
func (a *Accounts) RecordLogin(ctx context.Context, u *ent.User) error {
	return a.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		return recordUserAction(ctx, tx, pub, u, events.ActionUserLogin, nil)
	})
}

// recordUserAction publishes one Audit entry per Workspace the User holds a
// Membership in, and none anywhere else. The User is the actor and the target.
func recordUserAction(ctx context.Context, tx *ent.Client, pub events.Publisher, u *ent.User, action string, diff map[string]any) error {
	memberships, err := tx.Membership.Query().Where(membership.UserID(u.ID)).All(ctx)
	if err != nil {
		return err
	}
	id := strconv.FormatInt(u.ID, 10)
	for _, m := range memberships {
		if err := events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: m.WorkspaceID,
			Actor:       events.Actor{Kind: events.ActorUser, ID: id, Name: u.Name},
			Action:      action,
			TargetType:  "user",
			TargetID:    id,
			TargetName:  u.Name,
			Diff:        diff,
		}); err != nil {
			return err
		}
	}
	return nil
}

// MarkEmailVerified stamps the User's email as verified now.
func (a *Accounts) MarkEmailVerified(ctx context.Context, id int64) error {
	return a.ent.User.UpdateOneID(id).SetEmailVerifiedAt(time.Now()).Exec(ctx)
}

// ChangeEmail swaps the login email for one proven by a confirmation link, so it is
// stored verified. The unique index guards a race on the address (see
// db.IsUniqueViolation).
func (a *Accounts) ChangeEmail(ctx context.Context, id int64, newEmail string) error {
	return a.ent.User.UpdateOneID(id).SetEmail(newEmail).SetEmailVerifiedAt(time.Now()).Exec(ctx)
}

// PendingInvitation looks up a pending, unexpired Invitation by its raw token, with
// its Workspace edge. The token is the authorization, so no Workspace is known up
// front. An ent NotFound error when there is none.
func (a *Accounts) PendingInvitation(ctx context.Context, token string) (*ent.Invitation, error) {
	return a.ent.Invitation.Query().
		Where(
			invitation.TokenHash(HashInviteToken(token)),
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
	return a.bus.WithinTx(ctx, func(tx *ent.Client, pub events.Publisher) error {
		u, uerr := tx.User.Query().Where(user.Email(inv.Email)).Only(ctx)
		if ent.IsNotFound(uerr) {
			hash, herr := credentials.HashPassword(password)
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
			Save(ctx); merr != nil && !db.IsUniqueViolation(merr) {
			return merr
		}

		if err := ts.Invitation().UpdateOneID(inv.ID).SetAcceptedAt(time.Now()).Exec(ctx); err != nil {
			return err
		}
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: inv.WorkspaceID,
			Actor:       events.Actor{Kind: events.ActorUser, ID: strconv.FormatInt(u.ID, 10), Name: u.Name},
			Action:      events.ActionInvitationAccept,
			TargetType:  "invitation",
			TargetID:    strconv.FormatInt(inv.ID, 10),
			TargetName:  inv.Email,
			Diff:        map[string]any{"role": map[string]any{"to": string(inv.Role)}},
		})
	})
}

// InviteInput is a new (or reissued) Invitation request. The token, its hash and
// the expiry are the Invite operation's to decide, not the caller's.
type InviteInput struct {
	Email     string
	Role      invitation.Role
	InvitedBy int64
}

// Invite creates the Invitation for an email, or reissues the pending one (token,
// expiry, role; a prior acceptance is cleared), and records `invitation.create` in
// the same transaction. It generates the one-time token, stores only its hash, sets
// the expiry InviteTokenTTL ahead and returns the raw token for the caller's link;
// the token itself is never logged. ErrEmailEmpty for a blank email, ErrAlreadyMember
// when the address already has a Membership in the Workspace. Delivery is the
// caller's concern.
func (a *Accounts) Invite(ctx context.Context, s *ent.Scoped, actor events.Actor, in InviteInput) (*ent.Invitation, string, error) {
	email := strings.TrimSpace(in.Email)
	if email == "" {
		return nil, "", ErrEmailEmpty
	}
	token, err := generateInviteToken()
	if err != nil {
		return nil, "", err
	}
	tokenHash := HashInviteToken(token)
	expiresAt := time.Now().Add(InviteTokenTTL)

	var inv *ent.Invitation
	err = a.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, pub events.Publisher) error {
		member, err := ts.Membership().Query().
			Where(membership.HasUserWith(user.Email(email))).
			Exist(ctx)
		if err != nil {
			return err
		}
		if member {
			return ErrAlreadyMember
		}
		existing, err := ts.Invitation().Query().Where(invitation.Email(email)).Only(ctx)
		switch {
		case ent.IsNotFound(err):
			inv, err = ts.Invitation().Create().
				SetEmail(email).SetRole(in.Role).SetTokenHash(tokenHash).
				SetExpiresAt(expiresAt).SetInvitedBy(in.InvitedBy).
				Save(ctx)
		case err != nil:
		default:
			inv, err = ts.Invitation().UpdateOneID(existing.ID).
				SetRole(in.Role).SetTokenHash(tokenHash).
				SetExpiresAt(expiresAt).SetInvitedBy(in.InvitedBy).
				ClearAcceptedAt().
				Save(ctx)
		}
		if err != nil {
			return err
		}
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: s.WorkspaceID(),
			Actor:       actor,
			Action:      events.ActionInvitationCreate,
			TargetType:  "invitation",
			TargetID:    strconv.FormatInt(inv.ID, 10),
			TargetName:  inv.Email,
			Diff:        map[string]any{"role": map[string]any{"to": string(inv.Role)}},
		})
	})
	if err != nil {
		return nil, "", err
	}
	return inv, token, nil
}

// RevokeInvitation deletes a pending Invitation and records `invitation.revoke`. It
// reports whether the Invitation existed in the scoped Workspace.
func (a *Accounts) RevokeInvitation(ctx context.Context, s *ent.Scoped, actor events.Actor, id int64) (bool, error) {
	found := false
	err := a.bus.WithinScopedTx(ctx, s, func(ts *ent.Scoped, pub events.Publisher) error {
		inv, err := ts.Invitation().Query().Where(invitation.ID(id)).Only(ctx)
		if ent.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := ts.Invitation().DeleteOneID(id).Exec(ctx); err != nil {
			return err
		}
		found = true
		return events.RecordAudit(ctx, pub, &events.AuditEntry{
			WorkspaceID: s.WorkspaceID(),
			Actor:       actor,
			Action:      events.ActionInvitationRevoke,
			TargetType:  "invitation",
			TargetID:    strconv.FormatInt(id, 10),
			TargetName:  inv.Email,
		})
	})
	return found, err
}
