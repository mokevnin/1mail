package auth_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	"github.com/mokevnin/1mail/ent/apitoken"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/api/auth"
	"github.com/mokevnin/1mail/internal/credentials"
	"github.com/mokevnin/1mail/internal/db"
	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
)

// closedClient is an ent client whose database is gone: every query fails.
func closedClient(t *testing.T) *ent.Client {
	t.Helper()
	sqlDB, err := sql.Open("pgx", "postgres://closed.invalid/none")
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())
	return db.NewEntClient(sqlDB)
}

func TestHasScopeAndWorkspaceIDOfTheTokenContext(t *testing.T) {
	a := &auth.TokenAuth{TokenID: 7, WorkspaceID: 3, Scopes: []string{"contacts:read", "events:write"}}

	assert.True(t, auth.HasScope(a, "contacts:read"))
	assert.True(t, auth.HasScope(a, "events:write"))
	assert.False(t, auth.HasScope(a, "contacts:write"), "a read scope does not imply write")
	assert.False(t, auth.HasScope(a, ""))
	assert.False(t, auth.HasScope(nil, "contacts:read"), "no token has no scopes")

	assert.EqualValues(t, 3, a.WorkspaceID)
	assert.Nil(t, auth.TokenScoped(context.Background()), "unauthenticated has no scope, never a real tenant")
}

func TestAuthContextsRoundTrip(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, auth.GetTokenAuth(ctx))
	assert.Nil(t, auth.GetCollectAuth(ctx))
	assert.Nil(t, auth.GetSiteAuth(ctx))
	assert.Zero(t, collectWorkspaceID(ctx))

	token := &auth.TokenAuth{TokenID: 1, WorkspaceID: 2}
	assert.Same(t, token, auth.GetTokenAuth(auth.WithTokenAuth(ctx, token)))

	collect := &auth.CollectAuth{WorkspaceID: 5}
	withCollect := auth.WithCollectAuth(ctx, collect)
	assert.Same(t, collect, auth.GetCollectAuth(withCollect))
	assert.EqualValues(t, 5, collectWorkspaceID(withCollect))

	site := &auth.SiteAuth{UserID: 9, Email: "u@example.com"}
	assert.Same(t, site, auth.GetSiteAuth(auth.WithSiteAuth(ctx, site)))

	// The three identities live under distinct keys: one never reads as another.
	assert.Nil(t, auth.GetTokenAuth(withCollect))
	assert.Nil(t, auth.GetSiteAuth(withCollect))
}

func TestBearerAuthResolvesAValidTokenToItsWorkspaceAndScopes(t *testing.T) {
	env := testhelper.Setup(t)
	h := auth.NewExternalSecurityHandler(env.DB, env.Bus)

	ctx, err := h.HandleBearerAuth(context.Background(), "", externalapi.BearerAuth{
		Token: credentials.TokenValue(fixtures.AnchorTokenPrefix, fixtures.AnchorTokenSecret),
	})
	require.NoError(t, err)
	got := auth.GetTokenAuth(ctx)
	require.NotNil(t, got)
	assert.EqualValues(t, fixtures.AnchorTokenID, got.TokenID)
	assert.EqualValues(t, fixtures.AcmeID, got.WorkspaceID)
	assert.Equal(t, fixtures.AnchorTokenName, got.Name)
	assert.Contains(t, got.Scopes, "contacts:read")

	scoped := env.ScopedBearerFor(t, fixtures.GlobexID, "events:read")
	ctx, err = h.HandleBearerAuth(context.Background(), "", externalapi.BearerAuth{Token: scoped})
	require.NoError(t, err)
	assert.EqualValues(t, fixtures.GlobexID, auth.GetTokenAuth(ctx).WorkspaceID, "the token's own workspace, whichever it is")
	assert.Equal(t, []string{"events:read"}, auth.GetTokenAuth(ctx).Scopes)
}

func TestBearerAuthRejectsEveryUnusableToken(t *testing.T) {
	env := testhelper.Setup(t)
	h := auth.NewExternalSecurityHandler(env.DB, env.Bus)
	ctx := context.Background()
	valid := credentials.TokenValue(fixtures.AnchorTokenPrefix, fixtures.AnchorTokenSecret)

	revoked := env.ScopedBearer(t, "contacts:read")
	parsed := credentials.ParseToken(revoked)
	require.NotNil(t, parsed)
	n, err := env.DB.ApiToken.Update().Where(apitoken.Prefix(parsed.Prefix)).SetRevokedAt(time.Now()).Save(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	expired := env.ScopedBearer(t, "contacts:read")
	parsed = credentials.ParseToken(expired)
	require.NotNil(t, parsed)
	n, err = env.DB.ApiToken.Update().Where(apitoken.Prefix(parsed.Prefix)).SetExpiresAt(time.Now().Add(-time.Minute)).Save(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	for name, token := range map[string]string{
		"empty":            "",
		"malformed":        "not-a-token",
		"unknown prefix":   credentials.TokenValue("nosuchprefix", "whatever"),
		"wrong secret":     credentials.TokenValue(fixtures.AnchorTokenPrefix, "not-the-secret"),
		"revoked":          revoked,
		"expired":          expired,
		"valid prefix cut": valid[:len(valid)-len(fixtures.AnchorTokenSecret)-1],
	} {
		got, err := h.HandleBearerAuth(ctx, "", externalapi.BearerAuth{Token: token})
		require.ErrorIs(t, err, auth.ErrUnauthorized, name)
		assert.Nil(t, auth.GetTokenAuth(got), name)
	}
}

func TestBearerAuthStorageFailureIsNotAnUnauthorizedAnswer(t *testing.T) {
	h := auth.NewExternalSecurityHandler(closedClient(t), nil)
	_, err := h.HandleBearerAuth(context.Background(), "", externalapi.BearerAuth{
		Token: credentials.TokenValue(fixtures.AnchorTokenPrefix, fixtures.AnchorTokenSecret),
	})
	require.Error(t, err)
	assert.NotErrorIs(t, err, auth.ErrUnauthorized, "a database outage is a 500, not a 401 for a valid token")
}

func TestCollectKeyResolvesTheOwningWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	h := auth.NewCollectSecurityHandler(env.DB)

	for key, ws := range map[string]int64{fixtures.AcmeCollectKey: fixtures.AcmeID, fixtures.GlobexCollectKey: fixtures.GlobexID} {
		ctx, err := h.HandleApiKeyAuth(context.Background(), "", collectapi.ApiKeyAuth{APIKey: key})
		require.NoError(t, err)
		assert.EqualValues(t, ws, collectWorkspaceID(ctx))
	}

	for name, key := range map[string]string{"empty": "", "unknown": "omck_nope", "ingest key is not a collect key": fixtures.AcmeIngestKey} {
		ctx, err := h.HandleApiKeyAuth(context.Background(), "", collectapi.ApiKeyAuth{APIKey: key})
		require.ErrorIs(t, err, auth.ErrUnauthorized, name)
		assert.Zero(t, collectWorkspaceID(ctx), name)
	}

	_, err := auth.NewCollectSecurityHandler(closedClient(t)).HandleApiKeyAuth(context.Background(), "", collectapi.ApiKeyAuth{APIKey: fixtures.AcmeCollectKey})
	require.Error(t, err)
	assert.NotErrorIs(t, err, auth.ErrUnauthorized)
}

func siteJWT(t *testing.T, claims gptoken.Claims) string {
	t.Helper()
	cfg, err := config.Load("test")
	require.NoError(t, err)
	svc := gptoken.NewService(gptoken.Opts{
		SecretReader: gptoken.SecretFunc(func(string) (string, error) { return cfg.JWTSecret, nil }),
		Issuer:       "1mail",
		DisableXSRF:  true,
	})
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Issuer: "1mail", Audience: jwt.ClaimStrings{"1mail"}, ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	tk, err := svc.Token(claims)
	require.NoError(t, err)
	return tk
}

func TestSiteAuthResolvesTheDashboardUserFromTheJWT(t *testing.T) {
	env := testhelper.Setup(t)
	cfg, err := config.Load("test")
	require.NoError(t, err)
	h := auth.NewSiteSecurityHandler(cfg.JWTSecret, env.DB)

	// The direct provider stores the login (email) in User.Name.
	ctx, err := h.HandleApiKeyAuth(context.Background(), "", siteapi.ApiKeyAuth{
		APIKey: siteJWT(t, gptoken.Claims{User: &gptoken.User{Name: fixtures.OwnerJohnEmail, ID: "x"}}),
	})
	require.NoError(t, err)
	got := auth.GetSiteAuth(ctx)
	require.NotNil(t, got)
	assert.EqualValues(t, fixtures.OwnerJohnID, got.UserID)
	assert.Equal(t, fixtures.OwnerJohnEmail, got.Email)

	// Falls back to the Email claim when the login is empty.
	ctx, err = h.HandleApiKeyAuth(context.Background(), "", siteapi.ApiKeyAuth{
		APIKey: siteJWT(t, gptoken.Claims{User: &gptoken.User{Email: fixtures.OwnerJaneEmail, ID: "x"}}),
	})
	require.NoError(t, err)
	assert.EqualValues(t, fixtures.OwnerJaneID, auth.GetSiteAuth(ctx).UserID)
}

func TestSiteAuthRejectsEveryUnusableCredential(t *testing.T) {
	env := testhelper.Setup(t)
	cfg, err := config.Load("test")
	require.NoError(t, err)
	h := auth.NewSiteSecurityHandler(cfg.JWTSecret, env.DB)
	other := auth.NewSiteSecurityHandler("a-different-secret", env.DB)
	good := siteJWT(t, gptoken.Claims{User: &gptoken.User{Name: fixtures.OwnerJohnEmail, ID: "x"}})

	for name, tc := range map[string]struct {
		handler *auth.SiteSecurityHandler
		token   string
	}{
		"empty":             {h, ""},
		"garbage":           {h, "not.a.jwt"},
		"signed elsewhere":  {other, good},
		"no user claim":     {h, siteJWT(t, gptoken.Claims{})},
		"unknown user":      {h, siteJWT(t, gptoken.Claims{User: &gptoken.User{Name: "ghost@nowhere.test", ID: "x"}})},
		"user without name": {h, siteJWT(t, gptoken.Claims{User: &gptoken.User{ID: "x"}})},
	} {
		ctx, err := tc.handler.HandleApiKeyAuth(context.Background(), "", siteapi.ApiKeyAuth{APIKey: tc.token})
		require.ErrorIs(t, err, auth.ErrUnauthorized, name)
		assert.Nil(t, auth.GetSiteAuth(ctx), name)
	}

	_, err = auth.NewSiteSecurityHandler(cfg.JWTSecret, closedClient(t)).HandleApiKeyAuth(context.Background(), "", siteapi.ApiKeyAuth{APIKey: good})
	require.Error(t, err)
	assert.NotErrorIs(t, err, auth.ErrUnauthorized)
}

func TestCredCheckerVerifiesLoginCredentials(t *testing.T) {
	env := testhelper.Setup(t)
	c := auth.NewCredChecker(env.DB, accounts.NewAttempts(env.DB))

	ok, err := c.Check(fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword)
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = c.Check(fixtures.OwnerJohnEmail, "wrong")
	require.NoError(t, err)
	assert.False(t, ok, "wrong password")

	ok, err = c.Check("ghost@nowhere.test", fixtures.OwnerJohnPassword)
	require.NoError(t, err)
	assert.False(t, ok, "unknown user")

	jane, err := env.DB.User.Get(context.Background(), fixtures.OwnerJaneID)
	require.NoError(t, err)
	_, err = env.DB.User.UpdateOne(jane).SetPasswordHash("").Save(context.Background())
	require.NoError(t, err)
	ok, err = c.Check(fixtures.OwnerJaneEmail, "")
	require.NoError(t, err)
	assert.False(t, ok, "a user without a password hash cannot log in, even with an empty password")

	_, err = auth.NewCredChecker(closedClient(t), accounts.NewAttempts(closedClient(t))).Check(fixtures.OwnerJohnEmail, fixtures.OwnerJohnPassword)
	require.Error(t, err)
}

// collectWorkspaceID is the resolved collect key's Workspace id, 0 when unauthenticated.
func collectWorkspaceID(ctx context.Context) int64 {
	if a := auth.GetCollectAuth(ctx); a != nil {
		return a.WorkspaceID
	}
	return 0
}
