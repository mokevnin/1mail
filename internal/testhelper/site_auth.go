package testhelper

import (
	"context"
	"testing"
	"time"

	gptoken "github.com/go-pkgz/auth/v2/token"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mokevnin/1mail/config"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/stretchr/testify/require"
)

// SiteClient returns a typed /site client authenticated as the fixture user with
// the given login email: it carries a JWT cookie minted the way go-pkgz/auth's
// direct provider does, signed with the test config's secret.
func (env *TestEnv) SiteClient(t *testing.T, email string) *siteapi.Client {
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

	c, err := siteapi.NewClient("http://local/site", cookieAuth{tk}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)
	return c
}

type cookieAuth struct{ token string }

func (s cookieAuth) ApiKeyAuth(context.Context, siteapi.OperationName) (siteapi.ApiKeyAuth, error) {
	return siteapi.ApiKeyAuth{APIKey: s.token}, nil
}
