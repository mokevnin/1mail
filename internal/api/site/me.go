package site

import (
	"context"
	"net/http"
	"strings"

	entuser "github.com/mokevnin/1mail/ent/user"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/authtoken"
	"github.com/mokevnin/1mail/internal/service"
)

// SiteUserGetMe returns the authenticated user's profile. Auth is enforced by
// the security handler, so SiteAuth is always present here.
func (h *Handlers) SiteUserGetMe(ctx context.Context) (*siteapi.SiteUserResource, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	u, err := h.ent.User.Get(ctx, a.UserID)
	if err != nil {
		return nil, err
	}
	return mapper.UserToResource(u), nil
}

// SiteUserUpdateMe updates the authenticated user's name and/or password. Email
// is the login identity and is not editable here. Changing the password
// requires the correct current password.
func (h *Handlers) SiteUserUpdateMe(ctx context.Context, req *siteapi.SiteUpdateMeInput) (siteapi.SiteUserUpdateMeRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		v := siteapi.SiteUserUpdateMeForbidden(problem(http.StatusForbidden, "unauthorized"))
		return &v, nil
	}
	u, err := h.ent.User.Get(ctx, a.UserID)
	if err != nil {
		return nil, err
	}

	upd := h.ent.User.UpdateOneID(u.ID)
	changed := false

	if name, ok := req.Name.Get(); ok {
		name = strings.TrimSpace(name)
		if name == "" {
			v := siteapi.SiteUserUpdateMeUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity,
				"name must not be empty",
				map[string][]string{"name": {"name must not be empty"}},
			))
			return &v, nil
		}
		upd = upd.SetName(name)
		changed = true
	}

	if newPassword, ok := req.NewPassword.Get(); ok && newPassword != "" {
		currentPassword, _ := req.CurrentPassword.Get()
		if currentPassword == "" {
			v := siteapi.SiteUserUpdateMeUnprocessableEntity(problemWithErrors(
				http.StatusUnprocessableEntity,
				"current password is required",
				map[string][]string{"currentPassword": {"current password is required"}},
			))
			return &v, nil
		}
		// Verify the current password the same way the direct login provider does.
		if u.PasswordHash == "" || !service.VerifyPassword(u.PasswordHash, currentPassword) {
			v := siteapi.SiteUserUpdateMeForbidden(problem(http.StatusForbidden, "current password is incorrect"))
			return &v, nil
		}
		hash, err := service.HashPassword(newPassword)
		if err != nil {
			return nil, err
		}
		upd = upd.SetPasswordHash(hash)
		changed = true
	}

	if changed {
		u, err = upd.Save(ctx)
		if err != nil {
			return nil, err
		}
	}

	return mapper.UserToResource(u), nil
}

// SiteUserEmailChange requests a change of the login email. It verifies the
// current password, rejects an address already in use, and emails a
// confirmation link to the NEW address — the change only takes effect once that
// link is confirmed (SiteAuthConfirmEmailChange). Returns 202.
func (h *Handlers) SiteUserEmailChange(ctx context.Context, req *siteapi.SiteEmailChangeInput) (siteapi.SiteUserEmailChangeRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		v := siteapi.SiteUserEmailChangeForbidden(problem(http.StatusForbidden, "unauthorized"))
		return &v, nil
	}
	u, err := h.ent.User.Get(ctx, a.UserID)
	if err != nil {
		return nil, err
	}

	newEmail := strings.TrimSpace(string(req.NewEmail))
	if newEmail == "" {
		v := siteapi.SiteUserEmailChangeUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"new email is required",
			map[string][]string{"newEmail": {"new email is required"}},
		))
		return &v, nil
	}
	if strings.EqualFold(newEmail, u.Email) {
		v := siteapi.SiteUserEmailChangeUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"new email must differ from the current one",
			map[string][]string{"newEmail": {"new email must differ from the current one"}},
		))
		return &v, nil
	}

	if u.PasswordHash == "" || !service.VerifyPassword(u.PasswordHash, req.CurrentPassword) {
		v := siteapi.SiteUserEmailChangeForbidden(problem(http.StatusForbidden, "current password is incorrect"))
		return &v, nil
	}

	// Reject an address already taken. The confirm step re-checks under the unique
	// index, so this is an early, friendly 409 rather than the sole guard.
	taken, err := h.ent.User.Query().Where(entuser.Email(newEmail)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if taken {
		v := siteapi.SiteUserEmailChangeConflict(problem(http.StatusConflict, "that email is already in use"))
		return &v, nil
	}

	// Bind to the current email so the token is single-use (invalid once swapped);
	// carry the requested new address in the token.
	token, err := h.tokens.Mint(authtoken.PurposeEmailChange, u.ID, u.Email, emailChangeTokenTTL, map[string]string{"new": newEmail})
	if err != nil {
		return nil, err
	}
	_ = h.sysmail.EnqueueEmailChangeConfirm(ctx, newEmail, token)

	return &siteapi.SiteUserEmailChangeAccepted{}, nil
}

// SiteUserResendVerification re-sends the signup verification link to the
// current address. A no-op (still 202) when the email is already verified.
func (h *Handlers) SiteUserResendVerification(ctx context.Context) error {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return auth.ErrUnauthorized
	}
	u, err := h.ent.User.Get(ctx, a.UserID)
	if err != nil {
		return err
	}
	if u.EmailVerifiedAt != nil {
		return nil
	}
	token, err := h.tokens.Mint(authtoken.PurposeEmailVerify, u.ID, "", verifyTokenTTL, map[string]string{"email": u.Email})
	if err != nil {
		return err
	}
	_ = h.sysmail.EnqueueEmailVerification(ctx, u.Email, token)
	return nil
}
