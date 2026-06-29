package site

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/service"
)

func (h *Handlers) SiteAuthRegister(ctx context.Context, req *siteapi.SiteRegisterInput) (siteapi.SiteAuthRegisterRes, error) {
	name := strings.TrimSpace(req.Name)
	email := strings.TrimSpace(string(req.Email))
	password := req.Password

	// Required-field validation is hand-rolled here (and likewise in
	// SiteTokensCreate, SiteWorkspacesUpdate, SiteUserUpdateMe) rather than
	// declared as TypeSpec @minLength on the input models. ogen enforces spec
	// constraints in the request decoder *before* the handler runs, which would
	// collapse these into a generic 400 — untyped in the generated client,
	// since these operations only declare 422 — and drop the per-field error
	// map below. Hand-rolling also lets us TrimSpace and reject blank /
	// whitespace-only values, which @minLength(1) would accept. Keep it here.
	if name == "" || email == "" || password == "" {
		fieldErrors := map[string][]string{}
		if name == "" {
			fieldErrors["name"] = []string{"name is required"}
		}
		if email == "" {
			fieldErrors["email"] = []string{"email is required"}
		}
		if password == "" {
			fieldErrors["password"] = []string{"password is required"}
		}
		v := siteapi.SiteAuthRegisterUnprocessableEntity(problemWithErrors(
			http.StatusUnprocessableEntity,
			"name, email and password are required",
			fieldErrors,
		))
		return &v, nil
	}

	hash, err := service.HashPassword(password)
	if err != nil {
		return nil, err
	}

	u, err := h.ent.User.Create().
		SetName(name).
		SetEmail(email).
		SetPasswordHash(hash).
		Save(ctx)
	if service.IsUniqueViolation(err) {
		v := siteapi.SiteAuthRegisterConflict(problemWithErrors(
			http.StatusConflict,
			"email already exists",
			map[string][]string{"email": {"email already exists"}},
		))
		return &v, nil
	}
	if err != nil {
		return nil, err
	}

	if _, err := h.createDefaultWorkspace(ctx, u.ID, name); err != nil {
		return nil, err
	}

	// Welcome email is a platform (transactional) send via the system sender — a
	// river job in prod, run inline in tests. Best-effort: never fail registration.
	_ = h.welcome.EnqueueWelcome(ctx, u.Email, u.Name)

	return &siteapi.SiteRegisterResult{
		ID:        siteapi.EntityId(strconv.FormatInt(u.ID, 10)),
		Name:      u.Name,
		Email:     siteapi.EmailAddress(u.Email),
		CreatedAt: siteapi.Timestamp(u.CreatedAt),
	}, nil
}

// SiteAuthDirectLogin is unreachable in practice: the server mux routes
// /site/auth/direct/login to the go-pkgz/auth direct provider (see internal/server.New),
// which issues the JWT cookie. This stub only exists to satisfy the ogen Handler
// interface and acts as a defensive fallback if that route is ever removed.
func (h *Handlers) SiteAuthDirectLogin(_ context.Context, _ *siteapi.SiteDirectLoginInput) (siteapi.SiteAuthDirectLoginRes, error) {
	v := problem(http.StatusBadRequest, "direct login is handled by auth provider")
	return &v, nil
}
