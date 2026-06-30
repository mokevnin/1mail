package site_test

import (
	"context"
	"testing"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mokevnin/1mail/config"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// staticJWT supplies a fixed JWT cookie value as the site security source.
type staticJWT struct{ token string }

func (s staticJWT) ApiKeyAuth(context.Context, siteapi.OperationName) (siteapi.ApiKeyAuth, error) {
	return siteapi.ApiKeyAuth{APIKey: s.token}, nil
}

// jwtFor mints a JWT for the given login email, mirroring how go-pkgz/auth's
// direct provider issues tokens (login stored in User.Name), signed with the
// test config's JWT secret so SiteSecurityHandler accepts it.
func jwtFor(t *testing.T, email string) string {
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
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
		User: &gptoken.User{Name: email, ID: "test"},
	})
	require.NoError(t, err)
	return tk
}

func siteClient(t *testing.T, env *testhelper.TestEnv, email string) *siteapi.Client {
	t.Helper()
	c, err := siteapi.NewClient("http://local/site", staticJWT{jwtFor(t, email)}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

// Fixture user info@1mail.com owns workspace "acme" (id 1), which owns the three
// seeded contacts. The dashboard addresses contacts via /w/{slug}/contacts.
func TestSiteContactsScopedToWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	// Contacts of the owned workspace are listed.
	list, err := c.SiteContactsList(ctx, siteapi.SiteContactsListParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	listed, ok := list.(*siteapi.SiteContactsListOK)
	require.Truef(t, ok, "got %T", list)
	assert.Equal(t, int32(3), listed.TotalItems)

	// Creating a contact scopes it to the workspace.
	created, err := c.SiteContactsCreate(ctx, &siteapi.SiteCreateContactInput{Email: siteapi.NewOptNilEmailAddress("site-new@example.com")}, siteapi.SiteContactsCreateParams{WorkspaceSlug: "acme"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactResource{}, created)

	// An unknown / non-owned workspace slug resolves to 404, not a data leak.
	missing, err := c.SiteContactsList(ctx, siteapi.SiteContactsListParams{WorkspaceSlug: "does-not-exist"})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteContactsListNotFound{}, missing)
}

func TestSiteWorkspacesList(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	got, err := c.SiteWorkspacesList(context.Background())
	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "acme", got[0].Slug)
}
