package operator

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/mokevnin/sphericon/ent"
	operatorapi "github.com/mokevnin/sphericon/gen/operator"
	"github.com/mokevnin/sphericon/internal/i18n"
)

// Surface is the /operator API: the handlers and the security handler over one
// Module, for the composition root to mount.
type Surface struct {
	handlers *Handlers
	security *SecurityHandler
}

// NewSurface builds the surface.
func NewSurface(m *Module, s *Sessions) *Surface {
	return &Surface{handlers: &Handlers{module: m, sessions: s}, security: &SecurityHandler{sessions: s}}
}

// Server builds the ogen server under the /operator prefix. opts carry what the
// composition root owns (the RFC 7807 error handler).
func (s *Surface) Server(opts ...operatorapi.ServerOption) (http.Handler, error) {
	return operatorapi.NewServer(s.handlers, s.security,
		append([]operatorapi.ServerOption{operatorapi.WithPathPrefix("/operator")}, opts...)...)
}

// Handlers implements operatorapi.Handler.
type Handlers struct {
	module   *Module
	sessions *Sessions
}

var _ operatorapi.Handler = (*Handlers)(nil)

// OperatorAuthLogin is the password step: it answers a challenge (or, at first login,
// an enrolment) and never a session. An unknown email and a wrong password answer the
// same 401.
func (h *Handlers) OperatorAuthLogin(ctx context.Context, req *operatorapi.OperatorLoginInput) (operatorapi.OperatorAuthLoginRes, error) {
	op, err := h.module.CheckPassword(ctx, req.Email, req.Password)
	if errors.Is(err, ErrInvalidCredentials) {
		return unauthorized(i18n.T("errors.invalid_credentials", nil)), nil
	}
	if err != nil {
		return nil, err
	}
	step, err := h.module.StartSecondStep(ctx, op)
	if err != nil {
		return nil, err
	}
	res := &operatorapi.OperatorLoginResult{Outcome: operatorapi.OperatorLoginOutcomeChallenge, Challenge: step.Challenge}
	if step.Enrolment != nil {
		res.Outcome = operatorapi.OperatorLoginOutcomeEnrolment
		res.Enrolment = operatorapi.NewOptOperatorEnrolment(operatorapi.OperatorEnrolment{
			Secret: step.Enrolment.Secret, URI: step.Enrolment.URI, QrCode: step.Enrolment.QRCode,
		})
	}
	return res, nil
}

// OperatorAuthSecondFactor is the second step: a valid challenge and a current TOTP
// code start the session. A bad challenge and a wrong code answer the same 401.
func (h *Handlers) OperatorAuthSecondFactor(ctx context.Context, req *operatorapi.OperatorSecondFactorInput) (operatorapi.OperatorAuthSecondFactorRes, error) {
	op, err := h.module.Complete(ctx, req.Challenge, req.Code)
	if errors.Is(err, ErrInvalidChallenge) {
		return unauthorized(i18n.T("errors.login_challenge_invalid", nil)), nil
	}
	if errors.Is(err, ErrInvalidCode) {
		return unauthorized(i18n.T("errors.second_factor_code_invalid", nil)), nil
	}
	if err != nil {
		return nil, err
	}
	cookie, err := h.sessions.Issue(op)
	if err != nil {
		return nil, err
	}
	return &operatorapi.OperatorResourceHeaders{SetCookie: cookie.String(), Response: resource(op)}, nil
}

// OperatorAuthLogout ends the session on this browser by clearing its cookie. A token
// copied elsewhere stays valid until it expires or the Operator's epoch is bumped.
func (h *Handlers) OperatorAuthLogout(context.Context) (*operatorapi.OperatorAuthLogoutNoContent, error) {
	return &operatorapi.OperatorAuthLogoutNoContent{SetCookie: h.sessions.Cleared().String()}, nil
}

// OperatorMeGet is the current Operator.
func (h *Handlers) OperatorMeGet(ctx context.Context) (operatorapi.OperatorMeGetRes, error) {
	res := resource(Current(ctx))
	return &res, nil
}

func resource(op *ent.Operator) operatorapi.OperatorResource {
	return operatorapi.OperatorResource{ID: operatorapi.EntityId(strconv.FormatInt(op.ID, 10)), Email: operatorapi.EmailAddress(op.Email)}
}

// problem is the RFC 7807 body of a refusal.
type problem = operatorapi.ProblemDetails

// unauthorized is the 401 of a failed login step. It satisfies both login responses.
func unauthorized(detail string) *problem {
	return &problem{
		Status: operatorapi.NewOptInt32(http.StatusUnauthorized),
		Title:  operatorapi.NewOptString(http.StatusText(http.StatusUnauthorized)),
		Detail: operatorapi.NewOptString(detail),
	}
}
