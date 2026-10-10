package auth

import (
	"context"
	"errors"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	entuser "github.com/mokevnin/1mail/ent/user"
	"github.com/mokevnin/1mail/ent/workspace"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/ratelimit"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
)

// ErrUnauthorized is returned by security handlers when a request is not authorized.
var ErrUnauthorized = errors.New("unauthorized")

// TokenAuth holds the authenticated api-token context for external API requests.
type TokenAuth struct {
	TokenID     int64
	WorkspaceID int64
	Name        string
	Scopes      []string
	// Scoped is the client confined to WorkspaceID. This file is the external
	// API's construction point of it (ADR 0017); read it with TokenScoped.
	Scoped *ent.Scoped
}

type contextKey struct{}

var tokenAuthKey = contextKey{}

func WithTokenAuth(ctx context.Context, auth *TokenAuth) context.Context {
	return context.WithValue(ctx, tokenAuthKey, auth)
}

func GetTokenAuth(ctx context.Context) *TokenAuth {
	v, _ := ctx.Value(tokenAuthKey).(*TokenAuth)
	return v
}

func HasScope(auth *TokenAuth, scope string) bool {
	if auth == nil {
		return false
	}
	return lo.Contains(auth.Scopes, scope)
}

// TokenScoped returns the Workspace-scoped client of the authenticated api token.
// It is nil on an unauthenticated context, so call it only after the handler's
// HasScope check (which fails for a nil TokenAuth). Handlers pass it to the
// domain modules; they never build a *ent.Scoped from an id themselves.
func TokenScoped(ctx context.Context) *ent.Scoped {
	if a := GetTokenAuth(ctx); a != nil {
		return a.Scoped
	}
	return nil
}

// ExternalSecurityHandler implements externalapi.SecurityHandler (Bearer token auth).
type ExternalSecurityHandler struct {
	ent *ent.Client
}

func NewExternalSecurityHandler(client *ent.Client) *ExternalSecurityHandler {
	return &ExternalSecurityHandler{ent: client}
}

var _ externalapi.SecurityHandler = (*ExternalSecurityHandler)(nil)

// HandleBearerAuth authenticates the Bearer token and applies the rate limits that
// need it (ADR 0018): a client address that spent its failed-authentication budget
// is refused before the token is looked at, a failure counts against it, and a
// success charges the Workspace's shared /api and /mcp budget. Rejections surface
// as *ratelimit.LimitedError.
func (h *ExternalSecurityHandler) HandleBearerAuth(ctx context.Context, op externalapi.OperationName, t externalapi.BearerAuth) (context.Context, error) {
	limits := ratelimit.FromContext(ctx)
	if err := limits.AuthBlocked(); err != nil {
		return ctx, err
	}
	ctx, err := h.authenticate(ctx, op, t)
	if errors.Is(err, ErrUnauthorized) {
		if limited := limits.AuthFailed(); limited != nil {
			return ctx, limited
		}
		return ctx, err
	}
	if err != nil {
		return ctx, err
	}
	if err := limits.ChargeWorkspace(GetTokenAuth(ctx).WorkspaceID); err != nil {
		return ctx, err
	}
	return ctx, nil
}

func (h *ExternalSecurityHandler) authenticate(ctx context.Context, _ externalapi.OperationName, t externalapi.BearerAuth) (context.Context, error) {
	parsed := service.ParseToken(t.Token)
	if parsed == nil {
		return ctx, ErrUnauthorized
	}

	token, err := h.ent.ApiToken.Query().
		Where(apitoken.Prefix(parsed.Prefix)).
		First(ctx)
	if ent.IsNotFound(err) {
		return ctx, ErrUnauthorized
	}
	if err != nil {
		// Real DB error (timeout, pool exhaustion, ...) must surface as 500,
		// not masquerade as a 401 for an otherwise-valid token.
		return ctx, err
	}
	if token.RevokedAt != nil {
		return ctx, ErrUnauthorized
	}
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return ctx, ErrUnauthorized
	}
	if !service.VerifyTokenSecret(parsed.Secret, token.SecretHash) {
		return ctx, ErrUnauthorized
	}

	auth := &TokenAuth{
		TokenID:     token.ID,
		WorkspaceID: token.WorkspaceID,
		Name:        token.Name,
		Scopes:      token.Scopes,
		Scoped:      h.ent.Scoped(token.WorkspaceID),
	}
	return WithTokenAuth(ctx, auth), nil
}

// CollectAuth holds the workspace resolved from the per-workspace collect key.
type CollectAuth struct {
	WorkspaceID int64
	// Scoped is the client confined to WorkspaceID. This file is the collect
	// API's construction point of it (ADR 0017); read it with CollectScoped.
	Scoped *ent.Scoped
}

var collectAuthKey = struct{ name string }{"collectAuth"}

func WithCollectAuth(ctx context.Context, auth *CollectAuth) context.Context {
	return context.WithValue(ctx, collectAuthKey, auth)
}

func GetCollectAuth(ctx context.Context) *CollectAuth {
	v, _ := ctx.Value(collectAuthKey).(*CollectAuth)
	return v
}

// CollectScoped returns the Workspace-scoped client of the resolved collect key. It
// is nil on an unauthenticated context; the security handler runs before every
// collect operation, so handlers can rely on it.
func CollectScoped(ctx context.Context) *ent.Scoped {
	if a := GetCollectAuth(ctx); a != nil {
		return a.Scoped
	}
	return nil
}

// CollectSecurityHandler implements collectapi.SecurityHandler: resolves the
// per-workspace collect write-key (x-collect-key) to its workspace.
type CollectSecurityHandler struct {
	ent *ent.Client
}

func NewCollectSecurityHandler(client *ent.Client) *CollectSecurityHandler {
	return &CollectSecurityHandler{ent: client}
}

var _ collectapi.SecurityHandler = (*CollectSecurityHandler)(nil)

func (h *CollectSecurityHandler) HandleApiKeyAuth(ctx context.Context, _ collectapi.OperationName, t collectapi.ApiKeyAuth) (context.Context, error) {
	if t.APIKey == "" {
		return ctx, ErrUnauthorized
	}
	ws, err := h.ent.Workspace.Query().Where(workspace.CollectKey(t.APIKey)).Only(ctx)
	if ent.IsNotFound(err) {
		return ctx, ErrUnauthorized
	}
	if err != nil {
		return ctx, err
	}
	return WithCollectAuth(ctx, &CollectAuth{WorkspaceID: ws.ID, Scoped: h.ent.Scoped(ws.ID)}), nil
}

// SiteAuth holds the authenticated dashboard user resolved from the JWT cookie.
type SiteAuth struct {
	UserID int64
	Email  string
}

var siteAuthKey = struct{ name string }{"siteAuth"}

func WithSiteAuth(ctx context.Context, auth *SiteAuth) context.Context {
	return context.WithValue(ctx, siteAuthKey, auth)
}

func GetSiteAuth(ctx context.Context) *SiteAuth {
	v, _ := ctx.Value(siteAuthKey).(*SiteAuth)
	return v
}

// SiteSecurityHandler implements siteapi.SecurityHandler: validates the JWT
// cookie issued by go-pkgz/auth and resolves the dashboard user from it.
type SiteSecurityHandler struct {
	tokens *gptoken.Service
	ent    *ent.Client
}

func NewSiteSecurityHandler(jwtSecret string, client *ent.Client) *SiteSecurityHandler {
	svc := gptoken.NewService(gptoken.Opts{
		SecretReader: gptoken.SecretFunc(func(string) (string, error) { return jwtSecret, nil }),
		Issuer:       "1mail",
		DisableXSRF:  true,
	})
	return &SiteSecurityHandler{tokens: svc, ent: client}
}

var _ siteapi.SecurityHandler = (*SiteSecurityHandler)(nil)

func (h *SiteSecurityHandler) HandleApiKeyAuth(ctx context.Context, _ siteapi.OperationName, t siteapi.ApiKeyAuth) (context.Context, error) {
	claims, err := h.tokens.Parse(t.APIKey)
	if err != nil || claims.User == nil {
		return ctx, ErrUnauthorized
	}

	// The direct provider stores the login (email) in User.Name; resolve the ent user by it.
	email := claims.User.Name
	if email == "" {
		email = claims.User.Email
	}
	u, err := h.ent.User.Query().Where(entuser.Email(email)).Only(ctx)
	if ent.IsNotFound(err) {
		return ctx, ErrUnauthorized
	}
	if err != nil {
		return ctx, err
	}

	return WithSiteAuth(ctx, &SiteAuth{UserID: u.ID, Email: u.Email}), nil
}

// CredChecker verifies user credentials for go-pkgz/auth direct provider.
type CredChecker struct {
	ent *ent.Client
}

func NewCredChecker(client *ent.Client) *CredChecker {
	return &CredChecker{ent: client}
}

func (c *CredChecker) Check(user, password string) (bool, error) {
	u, err := c.ent.User.Query().Where(entuser.Email(user)).Only(context.Background())
	if ent.IsNotFound(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if u.PasswordHash == "" {
		return false, nil
	}
	return service.VerifyPassword(u.PasswordHash, password), nil
}
