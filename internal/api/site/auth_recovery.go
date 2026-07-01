package site

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/mokevnin/1mail/ent"
	entuser "github.com/mokevnin/1mail/ent/user"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/authtoken"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/service"
)

// Token lifetimes for the self-service account flows.
const (
	resetTokenTTL       = time.Hour
	verifyTokenTTL      = 24 * time.Hour
	emailChangeTokenTTL = time.Hour
)

// SiteAuthForgotPassword mints a reset token and emails a link. It always
// returns 202, whether or not the address matches an account, so the endpoint
// cannot be used to enumerate registered emails. (Rate limiting is deferred.)
func (h *Handlers) SiteAuthForgotPassword(ctx context.Context, req *siteapi.SiteForgotPasswordInput) error {
	email := strings.TrimSpace(string(req.Email))
	if email == "" {
		return nil
	}
	u, err := h.ent.User.Query().Where(entuser.Email(email)).Only(ctx)
	if ent.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Bind the token to the current password hash: once the password is reset the
	// hash changes and the token stops verifying (single use, no extra column).
	token, err := h.tokens.Mint(authtoken.PurposePasswordReset, u.ID, u.PasswordHash, resetTokenTTL, nil)
	if err != nil {
		return err
	}
	// Best-effort send (mirrors the welcome email): never fail the request.
	_ = h.sysmail.EnqueuePasswordReset(ctx, u.Email, token)
	return nil
}

// SiteAuthResetPassword sets a new password from a reset token.
func (h *Handlers) SiteAuthResetPassword(ctx context.Context, req *siteapi.SiteResetPasswordInput) (siteapi.SiteAuthResetPasswordRes, error) {
	if req.Password == "" {
		v := problem(http.StatusBadRequest, i18n.T("errors.password_required", nil))
		return &v, nil
	}
	uid, _, err := h.tokens.Parse(req.Token, authtoken.PurposePasswordReset, h.passwordHashBinding(ctx))
	if err != nil {
		v := problem(http.StatusBadRequest, i18n.T("errors.reset_link_invalid", nil))
		return &v, nil
	}
	hash, err := service.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	if err := h.ent.User.UpdateOneID(uid).SetPasswordHash(hash).Exec(ctx); err != nil {
		return nil, err
	}
	return &siteapi.SiteAuthResetPasswordOK{}, nil
}

// SiteAuthVerifyEmail confirms the address a token was minted for.
func (h *Handlers) SiteAuthVerifyEmail(ctx context.Context, req *siteapi.SiteVerifyEmailInput) (siteapi.SiteAuthVerifyEmailRes, error) {
	uid, extra, err := h.tokens.Parse(req.Token, authtoken.PurposeEmailVerify, noBinding)
	if err != nil {
		v := problem(http.StatusBadRequest, i18n.T("errors.verify_link_invalid", nil))
		return &v, nil
	}
	u, err := h.ent.User.Get(ctx, uid)
	if err != nil {
		v := problem(http.StatusBadRequest, i18n.T("errors.verify_link_bad", nil))
		return &v, nil
	}
	// A verify link is scoped to the address it was sent to; if the user has since
	// changed their email, an old link must not mark the new address verified.
	if claimed := extra["email"]; claimed != "" && claimed != u.Email {
		v := problem(http.StatusBadRequest, i18n.T("errors.verify_link_stale", nil))
		return &v, nil
	}
	if u.EmailVerifiedAt == nil {
		if err := h.ent.User.UpdateOneID(uid).SetEmailVerifiedAt(time.Now()).Exec(ctx); err != nil {
			return nil, err
		}
	}
	return &siteapi.SiteAuthVerifyEmailOK{}, nil
}

// SiteAuthConfirmEmailChange applies a requested email change once the new
// address is confirmed via the token sent to it. Public: the link is opened from
// the new inbox, which carries no session, and swapping the email invalidates any
// existing cookie anyway — the SPA routes the user to sign in with the new email.
func (h *Handlers) SiteAuthConfirmEmailChange(ctx context.Context, req *siteapi.SiteConfirmEmailChangeInput) (siteapi.SiteAuthConfirmEmailChangeRes, error) {
	uid, extra, err := h.tokens.Parse(req.Token, authtoken.PurposeEmailChange, h.currentEmailBinding(ctx))
	if err != nil {
		v := siteapi.SiteAuthConfirmEmailChangeBadRequest(problem(http.StatusBadRequest, i18n.T("errors.confirm_link_invalid", nil)))
		return &v, nil
	}
	newEmail := strings.TrimSpace(extra["new"])
	if newEmail == "" {
		v := siteapi.SiteAuthConfirmEmailChangeBadRequest(problem(http.StatusBadRequest, i18n.T("errors.confirm_link_bad", nil)))
		return &v, nil
	}
	// The new address was proven by clicking this link, so it is verified. The
	// unique index also guards the race if the address was taken meanwhile.
	err = h.ent.User.UpdateOneID(uid).SetEmail(newEmail).SetEmailVerifiedAt(time.Now()).Exec(ctx)
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteAuthConfirmEmailChangeConflict(problem(http.StatusConflict, i18n.T("errors.email_in_use", nil)))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteAuthConfirmEmailChangeOK{}, nil
}

// passwordHashBinding resolves a user's current password hash — the binding for
// reset tokens.
func (h *Handlers) passwordHashBinding(ctx context.Context) func(int64) (string, error) {
	return func(id int64) (string, error) {
		u, err := h.ent.User.Get(ctx, id)
		if err != nil {
			return "", err
		}
		return u.PasswordHash, nil
	}
}

// currentEmailBinding resolves a user's current email — the binding for
// email-change tokens.
func (h *Handlers) currentEmailBinding(ctx context.Context) func(int64) (string, error) {
	return func(id int64) (string, error) {
		u, err := h.ent.User.Get(ctx, id)
		if err != nil {
			return "", err
		}
		return u.Email, nil
	}
}

// noBinding is the binding for verify tokens, which are scoped by user id alone.
func noBinding(int64) (string, error) { return "", nil }
