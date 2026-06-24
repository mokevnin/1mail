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

// WorkspaceID returns the workspace the authenticated token belongs to (0 if unauthenticated).
func WorkspaceID(auth *TokenAuth) int64 {
	if auth == nil {
		return 0
	}
	return auth.WorkspaceID
}

// ExternalSecurityHandler implements externalapi.SecurityHandler (Bearer token auth).
type ExternalSecurityHandler struct {
	ent *ent.Client
}

func NewExternalSecurityHandler(client *ent.Client) *ExternalSecurityHandler {
	return &ExternalSecurityHandler{ent: client}
}

var _ externalapi.SecurityHandler = (*ExternalSecurityHandler)(nil)

func (h *ExternalSecurityHandler) HandleBearerAuth(ctx context.Context, _ externalapi.OperationName, t externalapi.BearerAuth) (context.Context, error) {
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
	}
	return WithTokenAuth(ctx, auth), nil
}

// CollectAuth holds the workspace resolved from the per-workspace collect key.
type CollectAuth struct {
	WorkspaceID int64
}

var collectAuthKey = struct{ name string }{"collectAuth"}

func WithCollectAuth(ctx context.Context, auth *CollectAuth) context.Context {
	return context.WithValue(ctx, collectAuthKey, auth)
}

func GetCollectAuth(ctx context.Context) *CollectAuth {
	v, _ := ctx.Value(collectAuthKey).(*CollectAuth)
	return v
}

// CollectWorkspaceID returns the workspace resolved from the collect key (0 if unauthenticated).
func CollectWorkspaceID(ctx context.Context) int64 {
	if a := GetCollectAuth(ctx); a != nil {
		return a.WorkspaceID
	}
	return 0
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
	return WithCollectAuth(ctx, &CollectAuth{WorkspaceID: ws.ID}), nil
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
