package auth

import (
	"context"
	"errors"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	entuser "github.com/mokevnin/1mail/ent/user"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/samber/lo"
	"golang.org/x/crypto/bcrypt"
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

// CollectSecurityHandler implements collectapi.SecurityHandler (x-collect-key).
type CollectSecurityHandler struct {
	key string
}

func NewCollectSecurityHandler(key string) *CollectSecurityHandler {
	return &CollectSecurityHandler{key: key}
}

var _ collectapi.SecurityHandler = (*CollectSecurityHandler)(nil)

func (h *CollectSecurityHandler) HandleApiKeyAuth(ctx context.Context, _ collectapi.OperationName, t collectapi.ApiKeyAuth) (context.Context, error) {
	if h.key == "" || t.APIKey != h.key {
		return ctx, ErrUnauthorized
	}
	return ctx, nil
}

// SiteSecurityHandler implements siteapi.SecurityHandler: validates the JWT
// cookie issued by go-pkgz/auth, using the same secret/issuer.
type SiteSecurityHandler struct {
	tokens *gptoken.Service
}

func NewSiteSecurityHandler(jwtSecret string) *SiteSecurityHandler {
	svc := gptoken.NewService(gptoken.Opts{
		SecretReader: gptoken.SecretFunc(func(string) (string, error) { return jwtSecret, nil }),
		Issuer:       "1mail",
		DisableXSRF:  true,
	})
	return &SiteSecurityHandler{tokens: svc}
}

var _ siteapi.SecurityHandler = (*SiteSecurityHandler)(nil)

func (h *SiteSecurityHandler) HandleApiKeyAuth(ctx context.Context, _ siteapi.OperationName, t siteapi.ApiKeyAuth) (context.Context, error) {
	if _, err := h.tokens.Parse(t.APIKey); err != nil {
		return ctx, ErrUnauthorized
	}
	return ctx, nil
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
	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil, nil
}
