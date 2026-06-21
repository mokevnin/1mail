package site_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loginStatus drives the real direct-login endpoint and returns the status code,
// used to prove a password change took effect end to end.
func loginStatus(t *testing.T, env *testhelper.TestEnv, email, password string) int {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/site/auth/direct/login",
		strings.NewReader(`{"user":"`+email+`","passwd":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	env.Server.ServeHTTP(rec, req)
	return rec.Code
}

func TestSiteUserGetMe(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	me, err := c.SiteUserGetMe(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "John", me.Name)
	assert.Equal(t, siteapi.EmailAddress("info@1mail.com"), me.Email)
}

func TestSiteUserUpdateMeName(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")
	ctx := context.Background()

	res, err := c.SiteUserUpdateMe(ctx, &siteapi.SiteUpdateMeInput{Name: siteapi.NewOptString("Renamed")})
	require.NoError(t, err)
	updated, ok := res.(*siteapi.SiteUserResource)
	require.Truef(t, ok, "got %T", res)
	assert.Equal(t, "Renamed", updated.Name)

	// The change persists.
	me, err := c.SiteUserGetMe(ctx)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", me.Name)
}

func TestSiteUserUpdateMeRejectsBlankName(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteUserUpdateMe(context.Background(),
		&siteapi.SiteUpdateMeInput{Name: siteapi.NewOptString("   ")})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserUpdateMeUnprocessableEntity{}, res)
}

func TestSiteUserUpdateMePasswordWrongCurrent(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteUserUpdateMe(context.Background(), &siteapi.SiteUpdateMeInput{
		CurrentPassword: siteapi.NewOptString("wrong"),
		NewPassword:     siteapi.NewOptString("newsecret123"),
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserUpdateMeForbidden{}, res)

	// Original password still works.
	assert.Equal(t, http.StatusOK, loginStatus(t, env, "info@1mail.com", "password"))
}

func TestSiteUserUpdateMePasswordChange(t *testing.T) {
	env := testhelper.Setup(t)
	c := siteClient(t, env, "info@1mail.com")

	res, err := c.SiteUserUpdateMe(context.Background(), &siteapi.SiteUpdateMeInput{
		CurrentPassword: siteapi.NewOptString("password"),
		NewPassword:     siteapi.NewOptString("newsecret123"),
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteUserResource{}, res)

	// The new password works; the old one no longer does.
	assert.Equal(t, http.StatusOK, loginStatus(t, env, "info@1mail.com", "newsecret123"))
	assert.Equal(t, http.StatusForbidden, loginStatus(t, env, "info@1mail.com", "password"))
}
