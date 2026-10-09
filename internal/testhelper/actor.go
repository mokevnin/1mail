package testhelper

import (
	"context"
	"testing"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mokevnin/1mail/config"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/service"
	ht "github.com/ogen-go/ogen/http"
	"github.com/stretchr/testify/require"
)

// An actor is a ready typed client for one API surface acting as one identity.
// JWT signing, token generation and hashing, security sources and the in-memory
// transport are internal to this file; tests only pick the identity.

// now is the single place the actors read the wall clock (JWT expiry). A future
// clock seam replaces it here without changing any actor signature.
func now() time.Time { return time.Now() }

const (
	siteBase     = "http://local/site"
	externalBase = "http://local/api"
	collectBase  = "http://local/collect"
)

// SiteActor returns a /site client authenticated as the fixture user with the
// given login email (use the fixtures.*Email constants).
func (env *TestEnv) SiteActor(t *testing.T, email string) *siteapi.Client {
	t.Helper()
	return env.siteClient(t, mintSiteJWT(t, email))
}

// SiteActorVia is SiteActor with the in-memory transport wrapped by wrap, for
// tests that observe the wire (e.g. raw response bodies).
func (env *TestEnv) SiteActorVia(t *testing.T, email string, wrap func(inner ht.Client) ht.Client) *siteapi.Client {
	t.Helper()
	c, err := siteapi.NewClient(siteBase, cookieSource{mintSiteJWT(t, email)}, siteapi.WithClient(wrap(env.Transport(nil))))
	require.NoError(t, err)
	return c
}

// SiteAnonymous returns a /site client carrying no credential.
func (env *TestEnv) SiteAnonymous(t *testing.T) *siteapi.Client {
	t.Helper()
	return env.siteClient(t, "")
}

// ExternalAnchor returns an /api client bearing the fixture anchor token
// (scopes contacts, tokens and events, read and write).
func (env *TestEnv) ExternalAnchor(t *testing.T) *externalapi.Client {
	t.Helper()
	return env.ExternalWithToken(t, service.TokenValue(fixtures.AnchorTokenPrefix, fixtures.AnchorTokenSecret))
}

// ExternalScoped returns an /api client bearing a fresh Acme token that has
// exactly the given scopes, created through the production token code.
func (env *TestEnv) ExternalScoped(t *testing.T, scopes ...string) *externalapi.Client {
	t.Helper()
	return env.ExternalWithToken(t, env.ScopedBearer(t, scopes...))
}

// ScopedBearer returns the raw bearer value of a fresh Acme token that has
// exactly the given scopes, for tests that must speak raw HTTP or MCP.
func (env *TestEnv) ScopedBearer(t *testing.T, scopes ...string) string {
	t.Helper()
	return env.ScopedBearerFor(t, fixtures.AcmeID, scopes...)
}

// ScopedBearerFor is ScopedBearer for another fixture Workspace (tenant-isolation
// tests, e.g. fixtures.GlobexID).
func (env *TestEnv) ScopedBearerFor(t *testing.T, workspaceID int64, scopes ...string) string {
	t.Helper()
	prefix, err := service.GenerateTokenPrefix()
	require.NoError(t, err)
	secret, err := service.GenerateTokenSecret()
	require.NoError(t, err)
	hash, err := service.HashTokenSecret(secret)
	require.NoError(t, err)
	_, err = env.DB.ApiToken.Create().
		SetName("actor-token").SetPrefix(prefix).SetSecretHash(hash).SetScopes(scopes).
		SetWorkspaceID(workspaceID).
		Save(t.Context())
	require.NoError(t, err)
	return service.TokenValue(prefix, secret)
}

// ExternalAnonymous returns an /api client carrying no credential.
func (env *TestEnv) ExternalAnonymous(t *testing.T) *externalapi.Client {
	t.Helper()
	return env.ExternalWithToken(t, "")
}

// ExternalWithToken returns an /api client bearing the given raw token value
// (for wrong-credential tests).
func (env *TestEnv) ExternalWithToken(t *testing.T, bearer string) *externalapi.Client {
	t.Helper()
	c, err := externalapi.NewClient(externalBase, bearerSource{bearer}, externalapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

// CollectAcme returns a /collect client carrying the Acme collect key.
func (env *TestEnv) CollectAcme(t *testing.T) *collectapi.Client {
	t.Helper()
	return env.CollectWithKey(t, fixtures.AcmeCollectKey)
}

// CollectAnonymous returns a /collect client carrying no key.
func (env *TestEnv) CollectAnonymous(t *testing.T) *collectapi.Client {
	t.Helper()
	return env.CollectWithKey(t, "")
}

// CollectWithKey returns a /collect client carrying the given raw key (for
// wrong-credential tests).
func (env *TestEnv) CollectWithKey(t *testing.T, key string) *collectapi.Client {
	t.Helper()
	c, err := collectapi.NewClient(collectBase, collectKeySource{key}, collectapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

func (env *TestEnv) siteClient(t *testing.T, jwtValue string) *siteapi.Client {
	t.Helper()
	c, err := siteapi.NewClient(siteBase, cookieSource{jwtValue}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

// mintSiteJWT signs a token the way go-pkgz/auth's direct provider does, with
// the test config's secret.
func mintSiteJWT(t *testing.T, email string) string {
	t.Helper()
	cfg, err := config.Load("test")
	require.NoError(t, err)

	svc := gptoken.NewService(gptoken.Opts{
		SecretReader: gptoken.SecretFunc(func(string) (string, error) { return cfg.JWTSecret, nil }),
		Issuer:       "1mail",
		DisableXSRF:  true,
	})
	tk, err := svc.Token(gptoken.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "1mail",
			Audience:  jwt.ClaimStrings{"1mail"},
			ExpiresAt: jwt.NewNumericDate(now().Add(time.Hour)),
		},
		User: &gptoken.User{Name: email, ID: "test"},
	})
	require.NoError(t, err)
	return tk
}

type cookieSource struct{ token string }

func (s cookieSource) ApiKeyAuth(context.Context, siteapi.OperationName) (siteapi.ApiKeyAuth, error) {
	return siteapi.ApiKeyAuth{APIKey: s.token}, nil
}

type bearerSource struct{ token string }

func (s bearerSource) BearerAuth(context.Context, externalapi.OperationName) (externalapi.BearerAuth, error) {
	return externalapi.BearerAuth{Token: s.token}, nil
}

type collectKeySource struct{ key string }

func (s collectKeySource) ApiKeyAuth(context.Context, collectapi.OperationName) (collectapi.ApiKeyAuth, error) {
	return collectapi.ApiKeyAuth{APIKey: s.key}, nil
}
