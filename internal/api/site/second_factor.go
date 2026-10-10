package site

import (
	"context"
	"errors"
	"net/http"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/i18n"
	"github.com/mokevnin/1mail/internal/secondfactor"
	"github.com/mokevnin/1mail/internal/service"
)

// SiteSecondFactorGetStatus answers whether the User has a Second factor, a pending
// enrollment, and how many Recovery codes are left (never the codes).
func (h *Handlers) SiteSecondFactorGetStatus(ctx context.Context) (*siteapi.SiteSecondFactorStatus, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	st, err := h.secondFactor.Status(ctx, a.UserID)
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSecondFactorStatus{
		Enabled: st.Enabled, Pending: st.Pending, RecoveryCodesRemaining: int32(st.RecoveryCodesRemaining),
	}, nil
}

// SiteSecondFactorStartEnrollment creates a pending TOTP secret (ADR 0020).
func (h *Handlers) SiteSecondFactorStartEnrollment(ctx context.Context) (siteapi.SiteSecondFactorStartEnrollmentRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	e, err := h.secondFactor.StartEnrollment(ctx, a.UserID)
	if errors.Is(err, secondfactor.ErrAlreadyActive) {
		v := problem(http.StatusConflict, i18n.T("errors.second_factor_already_active", nil))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSecondFactorEnrollment{Secret: e.Secret, OtpauthUri: e.URI, QrCode: e.QRCode}, nil
}

// SiteSecondFactorConfirmEnrollment activates the pending secret with a valid code,
// returns the Recovery codes once and reissues the acting session (the epoch moved).
func (h *Handlers) SiteSecondFactorConfirmEnrollment(ctx context.Context, req *siteapi.SiteSecondFactorConfirmInput) (siteapi.SiteSecondFactorConfirmEnrollmentRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	u, codes, err := h.secondFactor.ConfirmEnrollment(ctx, a.UserID, req.Code)
	switch {
	case errors.Is(err, secondfactor.ErrInvalidCode):
		v := siteapi.SiteSecondFactorConfirmEnrollmentUnprocessableEntity(codeInvalidProblem())
		return &v, nil
	case errors.Is(err, secondfactor.ErrNoPending):
		v := siteapi.SiteSecondFactorConfirmEnrollmentConflict(problem(http.StatusConflict, i18n.T("errors.second_factor_no_pending", nil)))
		return &v, nil
	case errors.Is(err, secondfactor.ErrAlreadyActive):
		v := siteapi.SiteSecondFactorConfirmEnrollmentConflict(problem(http.StatusConflict, i18n.T("errors.second_factor_already_active", nil)))
		return &v, nil
	case err != nil:
		return nil, err
	}
	cookie, err := h.sessions.Issue(u)
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteRecoveryCodesHeaders{SetCookie: cookie.String(), Response: siteapi.SiteRecoveryCodes{Codes: codes}}, nil
}

// SiteSecondFactorRegenerateRecoveryCodes replaces the Recovery codes after the
// password is proven, and reissues the acting session.
func (h *Handlers) SiteSecondFactorRegenerateRecoveryCodes(ctx context.Context, req *siteapi.SiteRecoveryCodesInput) (siteapi.SiteSecondFactorRegenerateRecoveryCodesRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	ok, err := h.passwordMatches(ctx, a.UserID, req.CurrentPassword)
	if err != nil {
		return nil, err
	}
	if !ok {
		v := siteapi.SiteSecondFactorRegenerateRecoveryCodesForbidden(problem(http.StatusForbidden, i18n.T("errors.current_password_incorrect", nil)))
		return &v, nil
	}
	u, codes, err := h.secondFactor.RegenerateRecoveryCodes(ctx, a.UserID)
	if errors.Is(err, secondfactor.ErrNotActive) {
		v := siteapi.SiteSecondFactorRegenerateRecoveryCodesConflict(problem(http.StatusConflict, i18n.T("errors.second_factor_not_active", nil)))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}
	cookie, err := h.sessions.Issue(u)
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteRecoveryCodesHeaders{SetCookie: cookie.String(), Response: siteapi.SiteRecoveryCodes{Codes: codes}}, nil
}

// SiteSecondFactorDisable removes the Second factor after the password and a
// current code are proven, and reissues the acting session.
func (h *Handlers) SiteSecondFactorDisable(ctx context.Context, req *siteapi.SiteSecondFactorDisableInput) (siteapi.SiteSecondFactorDisableRes, error) {
	a := auth.GetSiteAuth(ctx)
	if a == nil {
		return nil, auth.ErrUnauthorized
	}
	ok, err := h.passwordMatches(ctx, a.UserID, req.CurrentPassword)
	if err != nil {
		return nil, err
	}
	if !ok {
		v := siteapi.SiteSecondFactorDisableForbidden(problem(http.StatusForbidden, i18n.T("errors.current_password_incorrect", nil)))
		return &v, nil
	}
	u, err := h.secondFactor.Disable(ctx, a.UserID, req.Code)
	switch {
	case errors.Is(err, secondfactor.ErrInvalidCode):
		v := siteapi.SiteSecondFactorDisableUnprocessableEntity(codeInvalidProblem())
		return &v, nil
	case errors.Is(err, secondfactor.ErrNotActive):
		v := siteapi.SiteSecondFactorDisableConflict(problem(http.StatusConflict, i18n.T("errors.second_factor_not_active", nil)))
		return &v, nil
	case err != nil:
		return nil, err
	}
	cookie, err := h.sessions.Issue(u)
	if err != nil {
		return nil, err
	}
	return &siteapi.SiteSecondFactorDisableNoContent{SetCookie: cookie.String()}, nil
}

// passwordMatches checks the User's current password.
func (h *Handlers) passwordMatches(ctx context.Context, userID int64, password string) (bool, error) {
	u, err := h.accounts.User(ctx, userID)
	if err != nil {
		return false, err
	}
	return u.PasswordHash != "" && service.VerifyPassword(u.PasswordHash, password), nil
}

func codeInvalidProblem() siteapi.ProblemDetails {
	msg := i18n.T("errors.second_factor_code_invalid", nil)
	return problemWithErrors(http.StatusUnprocessableEntity, msg, map[string][]string{"code": {msg}})
}
