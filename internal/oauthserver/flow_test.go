package oauthserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"github.com/mokevnin/1mail/ent/apitoken"
	siteapi "github.com/mokevnin/1mail/gen/site"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/oauth2"
)

const owner = "info@1mail.com"

// consent plays the signed-in owner of workspace acme answering the consent screen.
func consent(t *testing.T, env *testhelper.TestEnv, in siteapi.SiteOAuthDecisionInput) *url.URL {
	t.Helper()
	res, err := env.SiteClient(t, owner).SiteOAuthDecide(t.Context(), &in)
	require.NoError(t, err)
	ok, isOK := res.(*siteapi.SiteOAuthDecisionResult)
	require.Truef(t, isOK, "got %T", res)
	u, err := url.Parse(ok.RedirectUrl)
	require.NoError(t, err)
	return u
}

func approval(scope string, allowSend bool) siteapi.SiteOAuthDecisionInput {
	return siteapi.SiteOAuthDecisionInput{
		ClientId:      fixtureClientID,
		RedirectUri:   fixtureRedirectURI,
		State:         siteapi.NewOptString("xyz"),
		CodeChallenge: oauth2.S256ChallengeFromVerifier(verifier),
		Scope:         siteapi.NewOptString(scope),
		WorkspaceSlug: "acme",
		Approve:       true,
		AllowSend:     siteapi.NewOptBool(allowSend),
	}
}

func exchange(t *testing.T, env *testhelper.TestEnv, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oauth/token", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return serve(env, req)
}

func tokenForm(code string) url.Values {
	return url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"client_id":     {fixtureClientID},
		"redirect_uri":  {fixtureRedirectURI},
		"code_verifier": {verifier},
	}
}

func TestConsentScreenDescribesTheRequest(t *testing.T) {
	env := testhelper.Setup(t)

	res, err := env.SiteClient(t, owner).SiteOAuthDescribe(t.Context(), siteapi.SiteOAuthDescribeParams{
		ClientId:    fixtureClientID,
		RedirectUri: fixtureRedirectURI,
		Scope:       siteapi.NewOptString("contacts:read emails:send bogus"),
	})
	require.NoError(t, err)
	got, ok := res.(*siteapi.SiteOAuthAuthorizationRequest)
	require.Truef(t, ok, "got %T", res)
	assert.Equal(t, "Fixture Connector", got.ClientName)
	assert.Equal(t, []string{"contacts:read"}, got.Scopes)
	assert.Equal(t, []string{"emails:send"}, got.SendScopes, "send-class scopes await an explicit opt-in")
}

func TestConsentScreenRejectsMismatchedRedirect(t *testing.T) {
	env := testhelper.Setup(t)

	res, err := env.SiteClient(t, owner).SiteOAuthDescribe(t.Context(), siteapi.SiteOAuthDescribeParams{
		ClientId:    fixtureClientID,
		RedirectUri: "https://evil.example/cb",
	})
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteOAuthDescribeBadRequest{}, res)
}

func TestConsentRequiresASession(t *testing.T) {
	env := testhelper.Setup(t)
	c, err := siteapi.NewClient("http://local/site", noAuth{}, siteapi.WithClient(env.Transport(nil)))
	require.NoError(t, err)

	_, err = c.SiteOAuthDescribe(t.Context(), siteapi.SiteOAuthDescribeParams{ClientId: fixtureClientID, RedirectUri: fixtureRedirectURI})
	assert.Error(t, err)
}

type noAuth struct{}

func (noAuth) ApiKeyAuth(context.Context, siteapi.OperationName) (siteapi.ApiKeyAuth, error) {
	return siteapi.ApiKeyAuth{}, nil
}

func TestDenyingConsentSendsAccessDeniedBack(t *testing.T) {
	env := testhelper.Setup(t)
	in := approval("", false)
	in.Approve = false

	back := consent(t, env, in)

	assert.Equal(t, "access_denied", back.Query().Get("error"))
	assert.Equal(t, "xyz", back.Query().Get("state"))
	assert.Empty(t, back.Query().Get("code"))
}

func TestConsentToAWorkspaceTheUserDoesNotBelongToIsRefused(t *testing.T) {
	env := testhelper.Setup(t)
	in := approval("", false)
	in.WorkspaceSlug = "someone-elses"

	res, err := env.SiteClient(t, owner).SiteOAuthDecide(t.Context(), &in)
	require.NoError(t, err)
	assert.IsType(t, &siteapi.SiteOAuthDecideNotFound{}, res)
}

func TestTokenEndpointIssuesAnOrdinaryScopedAPIToken(t *testing.T) {
	env := testhelper.Setup(t)
	back := consent(t, env, approval("contacts:read segments:read", false))
	require.Equal(t, "xyz", back.Query().Get("state"))

	rec := exchange(t, env, tokenForm(back.Query().Get("code")))

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
	var tok map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tok))
	assert.Equal(t, "Bearer", tok["token_type"])
	assert.Equal(t, "contacts:read segments:read", tok["scope"])

	// It is a normal workspace API token: stored scoped, and /mcp accepts it.
	row, err := env.DB.ApiToken.Query().Where(apitoken.Name("Fixture Connector (MCP)")).Only(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []string{"contacts:read", "segments:read"}, row.Scopes)
	assert.Equal(t, int64(1), row.WorkspaceID)
	assert.Nil(t, row.RevokedAt)

	session := env.MCPClient(t, tok["access_token"].(string))
	_, err = session.ListTools(t.Context(), nil)
	require.NoError(t, err)
}

func TestSendClassScopesNeedAnExplicitOptIn(t *testing.T) {
	env := testhelper.Setup(t)
	scope := "contacts:read emails:send"

	for allowSend, want := range map[bool][]string{
		false: {"contacts:read"},
		true:  {"contacts:read", "emails:send"},
	} {
		back := consent(t, env, approval(scope, allowSend))
		rec := exchange(t, env, tokenForm(back.Query().Get("code")))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var tok map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &tok))
		assert.Equal(t, strings.Join(want, " "), tok["scope"], "allowSend=%v", allowSend)
	}
}

func TestAuthorizationCodeIsSingleUse(t *testing.T) {
	env := testhelper.Setup(t)
	code := consent(t, env, approval("contacts:read", false)).Query().Get("code")

	first := exchange(t, env, tokenForm(code))
	second := exchange(t, env, tokenForm(code))

	assert.Equal(t, http.StatusOK, first.Code)
	assert.Equal(t, http.StatusBadRequest, second.Code)
	assert.Contains(t, second.Body.String(), "invalid_grant")
}

func TestTokenEndpointChecksPKCEAndTheOriginalRequest(t *testing.T) {
	env := testhelper.Setup(t)

	for name, mutate := range map[string]func(url.Values){
		"wrong verifier":         func(f url.Values) { f.Set("code_verifier", strings.Repeat("a", 43)) },
		"missing verifier":       func(f url.Values) { f.Del("code_verifier") },
		"different redirect":     func(f url.Values) { f.Set("redirect_uri", "https://connector.example/other") },
		"different client":       func(f url.Values) { f.Set("client_id", "someone-else") },
		"unsupported grant":      func(f url.Values) { f.Set("grant_type", "password") },
		"unknown code":           func(f url.Values) { f.Set("code", "nope") },
		"client credentials too": func(f url.Values) { f.Set("grant_type", "client_credentials") },
	} {
		t.Run(name, func(t *testing.T) {
			code := consent(t, env, approval("contacts:read", false)).Query().Get("code")
			form := tokenForm(code)
			mutate(form)

			rec := exchange(t, env, form)

			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.NotContains(t, rec.Body.String(), "access_token")
		})
	}
}

// roundTripper serves requests straight from the in-memory app.
type roundTripper struct{ env *testhelper.TestEnv }

func (r roundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	r.env.Server.ServeHTTP(rec, req)
	return rec.Result(), nil
}

// A standard MCP client (the official go-sdk) discovers the server from the 401
// challenge, registers itself, runs the PKCE flow through the consent screen and
// ends up connected to /mcp with an ordinary API token.
func TestStandardMCPClientConnectsThroughOAuth(t *testing.T) {
	env := testhelper.Setup(t)
	httpClient := &http.Client{Transport: roundTripper{env}}
	noFollow := &http.Client{
		Transport:     roundTripper{env},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	const callback = "https://agent.example/callback"

	handler, err := auth.NewAuthorizationCodeHandler(&auth.AuthorizationCodeHandlerConfig{
		DynamicClientRegistrationConfig: &auth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName:   "Test Agent",
				RedirectURIs: []string{callback},
			},
		},
		RedirectURL: callback,
		Client:      httpClient,
		// The user's browser: follow /oauth/authorize to the consent screen, then
		// approve as the signed-in owner and collect the code from the redirect.
		AuthorizationCodeFetcher: func(ctx context.Context, args *auth.AuthorizationArgs) (*auth.AuthorizationResult, error) {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.URL, nil)
			if err != nil {
				return nil, err
			}
			resp, err := noFollow.Do(req)
			if err != nil {
				return nil, err
			}
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusFound {
				t.Errorf("authorize answered %d", resp.StatusCode)
			}
			consentURL, err := url.Parse(resp.Header.Get("Location"))
			if err != nil {
				return nil, err
			}
			q := consentURL.Query()

			res, err := env.SiteClient(t, owner).SiteOAuthDecide(ctx, &siteapi.SiteOAuthDecisionInput{
				ClientId:      q.Get("client_id"),
				RedirectUri:   q.Get("redirect_uri"),
				State:         siteapi.NewOptString(q.Get("state")),
				CodeChallenge: q.Get("code_challenge"),
				Scope:         siteapi.NewOptString(q.Get("scope")),
				WorkspaceSlug: "acme",
				Approve:       true,
			})
			if err != nil {
				return nil, err
			}
			decision := res.(*siteapi.SiteOAuthDecisionResult)
			back, err := url.Parse(decision.RedirectUrl)
			if err != nil {
				return nil, err
			}
			return &auth.AuthorizationResult{Code: back.Query().Get("code"), State: back.Query().Get("state")}, nil
		},
	})
	require.NoError(t, err)

	transport := &mcp.StreamableClientTransport{
		Endpoint:             appURL(t) + "/mcp",
		HTTPClient:           httpClient,
		OAuthHandler:         handler,
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "test-agent", Version: "0"}, nil).Connect(t.Context(), transport, nil)
	require.NoError(t, err, "connect through OAuth")
	t.Cleanup(func() { _ = session.Close() })

	tools, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)
	assert.NotEmpty(t, tools.Tools)

	row, err := env.DB.ApiToken.Query().Where(apitoken.Name("Test Agent (MCP)")).Only(t.Context())
	require.NoError(t, err)
	assert.NotContains(t, row.Scopes, "emails:send", "send-class scopes are opt-in")
	assert.Contains(t, row.Scopes, "contacts:read")
}
