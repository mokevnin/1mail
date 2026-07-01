package site

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/invitation"
	"github.com/mokevnin/1mail/ent/membership"
	entuser "github.com/mokevnin/1mail/ent/user"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/service"
)

// inviteTokenTTL is how long an invitation link stays valid.
const inviteTokenTTL = 7 * 24 * time.Hour

// invitationResource projects an Invitation (optionally with its inviter edge)
// into the site DTO.
func invitationResource(inv *ent.Invitation) siteapi.SiteInvitationResource {
	invitedBy := siteapi.OptNilString{}
	if inv.Edges.Inviter != nil {
		invitedBy = siteapi.NewOptNilString(inv.Edges.Inviter.Email)
	}
	return siteapi.SiteInvitationResource{
		ID:             siteapi.EntityId(strconv.FormatInt(inv.ID, 10)),
		Email:          siteapi.EmailAddress(inv.Email),
		Role:           siteapi.SiteInvitableRole(inv.Role),
		ExpiresAt:      siteapi.Timestamp(inv.ExpiresAt),
		InvitedByEmail: invitedBy,
		CreatedAt:      siteapi.Timestamp(inv.CreatedAt),
	}
}

// SiteInvitationsList returns the workspace's pending (unaccepted) invitations.
func (h *Handlers) SiteInvitationsList(ctx context.Context, params siteapi.SiteInvitationsListParams) (siteapi.SiteInvitationsListRes, error) {
	ws, _, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "workspace not found")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	invites, err := h.ent.Invitation.Query().
		Where(invitation.WorkspaceID(ws), invitation.AcceptedAtIsNil()).
		WithInviter().
		Order(ent.Asc(invitation.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	items := make(siteapi.SiteInvitationsListOKApplicationJSON, len(invites))
	for i, inv := range invites {
		items[i] = invitationResource(inv)
	}
	return &items, nil
}

// SiteInvitationsCreate invites an email address to the workspace. Owner/admin
// only. The one-time accept link is returned (copy-link path) and an invite email
// is sent best-effort — a missing/failed mailer must not fail the invite.
func (h *Handlers) SiteInvitationsCreate(ctx context.Context, req *siteapi.SiteCreateInvitationInput, params siteapi.SiteInvitationsCreateParams) (siteapi.SiteInvitationsCreateRes, error) {
	ws, callerRole, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteInvitationsCreateNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !canManageMembers(callerRole) {
		v := siteapi.SiteInvitationsCreateForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	email := strings.TrimSpace(string(req.Email))
	if email == "" {
		v := siteapi.SiteInvitationsCreateUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity, "email must not be empty",
			map[string][]string{"email": {"email must not be empty"}}))
		return &v, nil
	}

	// Already a member? Nothing to invite.
	alreadyMember, err := h.ent.Membership.Query().
		Where(membership.WorkspaceID(ws), membership.HasUserWith(entuser.Email(email))).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if alreadyMember {
		v := siteapi.SiteInvitationsCreateConflict(problem(http.StatusConflict, "already a member"))
		return &v, nil
	}

	token, err := service.GenerateInviteToken()
	if err != nil {
		return nil, err
	}
	tokenHash := service.HashInviteToken(token)
	role := invitation.Role(req.Role)
	expiresAt := time.Now().Add(inviteTokenTTL)
	a := auth.GetSiteAuth(ctx)

	// Upsert on the (workspace, email) unique key: re-inviting reissues the token
	// and expiry and clears any prior acceptance.
	existing, err := h.ent.Invitation.Query().
		Where(invitation.WorkspaceID(ws), invitation.Email(email)).
		Only(ctx)
	var inv *ent.Invitation
	switch {
	case ent.IsNotFound(err):
		inv, err = h.ent.Invitation.Create().
			SetWorkspaceID(ws).
			SetEmail(email).
			SetRole(role).
			SetTokenHash(tokenHash).
			SetExpiresAt(expiresAt).
			SetInvitedBy(a.UserID).
			Save(ctx)
		if err != nil {
			return nil, err
		}
	case err != nil:
		return nil, err
	default:
		inv, err = existing.Update().
			SetRole(role).
			SetTokenHash(tokenHash).
			SetExpiresAt(expiresAt).
			SetInvitedBy(a.UserID).
			ClearAcceptedAt().
			Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	inviteURL := strings.TrimRight(h.appURL, "/") + "/invitations/" + token

	// Send the invite email best-effort: the copy-link above already delivered
	// the invite, so self-hosted instances without SMTP still work.
	wsEnt, err := h.ent.Workspace.Get(ctx, ws)
	if err != nil {
		return nil, err
	}
	inviterName := ""
	if caller, cerr := h.ent.User.Get(ctx, a.UserID); cerr == nil {
		inviterName = caller.Name
	}
	if merr := h.sysmail.EnqueueMemberInvite(ctx, email, inviteURL, wsEnt.Name, inviterName); merr != nil {
		slog.WarnContext(ctx, "member invite email not enqueued", "error", merr, "email", email)
	}

	inv.Edges.Inviter, _ = h.ent.User.Get(ctx, a.UserID)
	return &siteapi.SiteCreateInvitationResponse{
		InviteUrl: inviteURL,
		Resource:  invitationResource(inv),
	}, nil
}

// SiteInvitationsDelete revokes a pending invitation. Owner/admin only.
func (h *Handlers) SiteInvitationsDelete(ctx context.Context, params siteapi.SiteInvitationsDeleteParams) (siteapi.SiteInvitationsDeleteRes, error) {
	ws, callerRole, err := h.membershipFor(ctx, params.Slug)
	if ent.IsNotFound(err) {
		v := siteapi.SiteInvitationsDeleteNotFound(problem(http.StatusNotFound, "workspace not found"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	if !canManageMembers(callerRole) {
		v := siteapi.SiteInvitationsDeleteForbidden(problem(http.StatusForbidden, "insufficient role"))
		return &v, nil
	}

	id, err := strconv.ParseInt(string(params.ID), 10, 64)
	if err != nil {
		v := siteapi.SiteInvitationsDeleteNotFound(problem(http.StatusNotFound, "invitation not found"))
		return &v, nil
	}

	n, err := h.ent.Invitation.Delete().
		Where(invitation.ID(id), invitation.WorkspaceID(ws)).
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	if n == 0 {
		v := siteapi.SiteInvitationsDeleteNotFound(problem(http.StatusNotFound, "invitation not found"))
		return &v, nil
	}
	return &siteapi.SiteInvitationsDeleteNoContent{}, nil
}

// validInvitation looks up a pending, unexpired invitation by its raw token.
func (h *Handlers) validInvitation(ctx context.Context, token string) (*ent.Invitation, error) {
	return h.ent.Invitation.Query().
		Where(
			invitation.TokenHash(service.HashInviteToken(token)),
			invitation.AcceptedAtIsNil(),
			invitation.ExpiresAtGT(time.Now()),
		).
		WithWorkspace().
		Only(ctx)
}

// SitePublicInvitationsLookup reveals a pending invite (workspace + email) so the
// public accept page can render. Session-less: the token is the authorization.
func (h *Handlers) SitePublicInvitationsLookup(ctx context.Context, params siteapi.SitePublicInvitationsLookupParams) (siteapi.SitePublicInvitationsLookupRes, error) {
	inv, err := h.validInvitation(ctx, params.Token)
	if ent.IsNotFound(err) {
		v := problem(http.StatusNotFound, "invitation not found or expired")
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	hasAccount, err := h.ent.User.Query().Where(entuser.Email(inv.Email)).Exist(ctx)
	if err != nil {
		return nil, err
	}

	return &siteapi.SiteInvitationLookupResult{
		WorkspaceName: inv.Edges.Workspace.Name,
		Email:         siteapi.EmailAddress(inv.Email),
		HasAccount:    hasAccount,
	}, nil
}

// SitePublicInvitationsAccept turns an invite into a Membership: it attaches an
// existing User or creates one (name + password), then joins them to the
// workspace. Link possession is the authorization.
func (h *Handlers) SitePublicInvitationsAccept(ctx context.Context, req *siteapi.SiteAcceptInvitationInput, params siteapi.SitePublicInvitationsAcceptParams) (siteapi.SitePublicInvitationsAcceptRes, error) {
	inv, err := h.validInvitation(ctx, params.Token)
	if ent.IsNotFound(err) {
		v := siteapi.SitePublicInvitationsAcceptNotFound(problem(http.StatusNotFound, "invitation not found or expired"))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	name, _ := req.Name.Get()
	name = strings.TrimSpace(name)
	password, _ := req.Password.Get()

	userExists, err := h.ent.User.Query().Where(entuser.Email(inv.Email)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	// A new invitee must set up an account.
	if !userExists && (name == "" || password == "") {
		v := siteapi.SitePublicInvitationsAcceptUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity, "name and password are required",
			map[string][]string{
				"name":     {"name is required"},
				"password": {"password is required"},
			}))
		return &v, nil
	}

	role := membership.Role(inv.Role)
	err = h.bus.WithinTx(ctx, func(tx *ent.Client, _ events.Publisher) error {
		u, uerr := tx.User.Query().Where(entuser.Email(inv.Email)).Only(ctx)
		if ent.IsNotFound(uerr) {
			hash, herr := service.HashPassword(password)
			if herr != nil {
				return herr
			}
			// The invite link proves control of the address, so the new account
			// is created already email-verified.
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

		// Idempotent join: a duplicate (already a member) is not an error.
		if _, merr := tx.Membership.Create().
			SetUserID(u.ID).
			SetWorkspaceID(inv.WorkspaceID).
			SetRole(role).
			Save(ctx); merr != nil && !service.IsUniqueViolation(merr) {
			return merr
		}

		return tx.Invitation.UpdateOneID(inv.ID).SetAcceptedAt(time.Now()).Exec(ctx)
	})
	if err != nil {
		return nil, err
	}

	return &siteapi.SitePublicInvitationsAcceptOK{}, nil
}
