package site_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// jwtCookie returns the value of the "JWT" cookie set on the response, or "" if absent.
func jwtCookie(resp *http.Response) string {
	for _, c := range resp.Cookies() {
		if c.Name == "JWT" {
			return c.Value
		}
	}
	return ""
}

// TestSiteDirectLoginSetsJWTAndAuthorizes proves the full login chain: the SPA's
// /site/auth/direct/login is routed to the go-pkgz/auth direct provider (server.New),
// which validates the seed user, sets a JWT cookie, and that cookie is then accepted by
// SiteSecurityHandler on a protected endpoint. Seed user info@1mail.com / password owns
// workspace "acme" (see fixtures).
func TestSiteDirectLoginSetsJWTAndAuthorizes(t *testing.T) {
	env := testhelper.Setup(t)

	// 1. Log in. go-pkgz/auth's direct provider reads JSON {user, passwd}.
	loginRec := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/site/auth/direct/login",
		strings.NewReader(`{"user":"info@1mail.com","passwd":"password"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	env.Server.ServeHTTP(loginRec, loginReq)

	require.Equal(t, http.StatusOK, loginRec.Code, "body: %s", loginRec.Body.String())
	jwt := jwtCookie(loginRec.Result())
	require.NotEmpty(t, jwt, "login must set a non-empty JWT cookie")

	// 2. Replay the JWT cookie on a protected endpoint; it must authorize.
	meRec := httptest.NewRecorder()
	meReq := httptest.NewRequest(http.MethodGet, "/site/workspaces", nil)
	meReq.AddCookie(&http.Cookie{Name: "JWT", Value: jwt})
	env.Server.ServeHTTP(meRec, meReq)

	assert.Equal(t, http.StatusOK, meRec.Code, "body: %s", meRec.Body.String())
}

func TestSiteDirectLoginRejectsBadPassword(t *testing.T) {
	env := testhelper.Setup(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/site/auth/direct/login",
		strings.NewReader(`{"user":"info@1mail.com","passwd":"wrong"}`))
	req.Header.Set("Content-Type", "application/json")
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Empty(t, jwtCookie(rec.Result()), "no JWT cookie on failed login")
}

// TestSiteWorkspacesRequireAuth brackets the guard: no cookie => 401.
func TestSiteWorkspacesRequireAuth(t *testing.T) {
	env := testhelper.Setup(t)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/site/workspaces", nil)
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
