package operator

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/mokevnin/sphericon/ent"
	"github.com/mokevnin/sphericon/ent/workspace"
	operatorapi "github.com/mokevnin/sphericon/gen/operator"
	"github.com/mokevnin/sphericon/internal/i18n"
	"github.com/mokevnin/sphericon/internal/pagination"
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

// OperatorWorkspacesList is a page of Workspaces, newest first, narrowed by a slug
// search. Metadata only: no Contacts, content or Events.
func (h *Handlers) OperatorWorkspacesList(ctx context.Context, params operatorapi.OperatorWorkspacesListParams) (operatorapi.OperatorWorkspacesListRes, error) {
	page, err := h.module.ListWorkspaces(ctx, strings.TrimSpace(params.Slug.Value), pagination.ParamsOf(params.Page, params.PageSize))
	if err != nil {
		return nil, err
	}
	items := make([]operatorapi.OperatorWorkspaceResource, 0, len(page.Items))
	for _, w := range page.Items {
		items = append(items, workspaceResource(w))
	}
	return &operatorapi.OperatorWorkspacesListOK{
		Items:      items,
		Page:       int32(page.Page),
		PageSize:   int32(page.PageSize),
		TotalItems: int32(page.TotalItems),
		TotalPages: int32(page.TotalPages),
	}, nil
}

// OperatorWorkspacesGet is one Workspace's metadata and suspension state.
func (h *Handlers) OperatorWorkspacesGet(ctx context.Context, params operatorapi.OperatorWorkspacesGetParams) (operatorapi.OperatorWorkspacesGetRes, error) {
	id, err := strconv.ParseInt(string(params.WorkspaceId), 10, 64)
	if err != nil {
		return workspaceNotFound(), nil
	}
	w, err := h.module.GetWorkspace(ctx, id)
	if ent.IsNotFound(err) {
		return workspaceNotFound(), nil
	}
	if err != nil {
		return nil, err
	}
	res := workspaceResource(w)
	return &res, nil
}

func workspaceNotFound() *operatorapi.OperatorWorkspacesGetNotFound {
	p := operatorapi.OperatorWorkspacesGetNotFound(operatorapi.ProblemDetails{
		Status: operatorapi.NewOptInt32(http.StatusNotFound),
		Title:  operatorapi.NewOptString(http.StatusText(http.StatusNotFound)),
		Detail: operatorapi.NewOptString(i18n.T("errors.workspace_not_found", nil)),
	})
	return &p
}

// workspaceResource is the console's view of a Workspace: metadata and suspension
// state only. The Operator sees the real actor id, which a customer never does.
func workspaceResource(w *ent.Workspace) operatorapi.OperatorWorkspaceResource {
	res := operatorapi.OperatorWorkspaceResource{
		ID:        operatorapi.EntityId(strconv.FormatInt(w.ID, 10)),
		Slug:      w.Slug,
		Name:      w.Name,
		CreatedAt: operatorapi.Timestamp(w.CreatedAt),
	}
	if w.SuspendedAt == nil {
		return res
	}
	actor := operatorapi.OperatorSuspensionActor{}
	if w.SuspendedByKind != nil {
		actor.Kind = operatorapi.OperatorSuspensionActorKind(*w.SuspendedByKind)
	}
	if w.SuspendedByKind != nil && *w.SuspendedByKind == workspace.SuspendedByKindOperator && w.SuspendedByID != nil {
		actor.ID = operatorapi.NewOptNilString(*w.SuspendedByID)
	}
	s := operatorapi.OperatorSuspension{At: operatorapi.Timestamp(*w.SuspendedAt), Actor: actor}
	if w.SuspensionReason != nil {
		s.Reason = operatorapi.NewOptNilString(*w.SuspensionReason)
	}
	res.Suspension = operatorapi.NewOptNilOperatorSuspension(s)
	return res
}
