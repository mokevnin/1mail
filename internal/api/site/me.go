package site

import (
	"context"
	"net/http"
	"strings"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
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
