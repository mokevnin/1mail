package mcpserver_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/mokevnin/1mail/internal/service"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// seedToken inserts an ApiToken in the given workspace and returns the plaintext
// bearer value (the real token crypto is exercised).
func seedToken(t *testing.T, env *testhelper.TestEnv, workspaceID int, scopes []string) string {
	t.Helper()
	prefix, err := service.GenerateTokenPrefix()
	require.NoError(t, err)
	secret, err := service.GenerateTokenSecret()
	require.NoError(t, err)
	hash, err := service.HashTokenSecret(secret)
	require.NoError(t, err)
	_, err = env.DB.ApiToken.Create().
		SetName("mcp-test").SetPrefix(prefix).SetSecretHash(hash).SetScopes(scopes).
		SetWorkspaceID(int64(workspaceID)).
		Save(context.Background())
	require.NoError(t, err)
	return service.TokenValue(prefix, secret)
}

func call(t *testing.T, s *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := s.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)
	return res
}

func text(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.Len(t, res.Content, 1)
	tc, ok := res.Content[0].(*mcp.TextContent)
	require.Truef(t, ok, "got %T", res.Content[0])
	return tc.Text
}

func TestMCPRequiresBearerToken(t *testing.T) {
	env := testhelper.Setup(t)
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"x","version":"0"}}}`

	for name, auth := range map[string]string{
		"missing":   "",
		"malformed": "Bearer nonsense",
		"unknown":   "Bearer omtk_deadbeef_invalidsecret",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, testhelper.MCPEndpoint, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			if auth != "" {
				req.Header.Set("Authorization", auth)
			}
			rec := httptest.NewRecorder()
			env.Server.ServeHTTP(rec, req)
			assert.Equal(t, http.StatusUnauthorized, rec.Code)
		})
	}
}

func TestMCPToolsAreTheContractMinusHiddenOperations(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.MCPClient(t, seedToken(t, env, 1, []string{"contacts:read"}))

	list, err := s.ListTools(context.Background(), nil)
	require.NoError(t, err)

	got := map[string]*mcp.Tool{}
	for _, tool := range list.Tools {
		got[tool.Name] = tool
	}
	names := make([]string, 0, len(got))
	for n := range got {
		names = append(names, n)
	}
	// Independent literal: the /api operations that are not x-mcp hidden, by 1mail name.
	// Segments, broadcasts and token management are hidden.
	assert.ElementsMatch(t, []string{
		"contacts_list", "contacts_create", "contacts_get", "contacts_update", "contacts_delete",
		"emails_send", "events_record", "events_actions_list", "whoami",
		"suppressions_create", "unsubscribes_create",
		"tags_list", "tags_list_for_contact", "tags_apply", "tags_remove",
	}, names)

	// Hints derive from the HTTP method.
	ro := got["contacts_list"].Annotations
	require.NotNil(t, ro)
	assert.True(t, ro.ReadOnlyHint)

	del := got["contacts_delete"].Annotations
	require.NotNil(t, del)
	assert.False(t, del.ReadOnlyHint)
	require.NotNil(t, del.DestructiveHint)
	assert.True(t, *del.DestructiveHint)
	assert.True(t, del.IdempotentHint)

	post := got["contacts_create"].Annotations
	require.NotNil(t, post)
	require.NotNil(t, post.DestructiveHint)
	assert.False(t, *post.DestructiveHint)
	assert.False(t, post.IdempotentHint)

	put := got["contacts_update"].Annotations
	require.NotNil(t, put)
	assert.True(t, put.IdempotentHint)

	assert.NotEmpty(t, got["contacts_list"].Description)
}

func TestMCPToolCallReturnsTheAPIResult(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.MCPClient(t, seedToken(t, env, 1, []string{"contacts:read", "contacts:write"}))

	res := call(t, s, "contacts_get", map[string]any{"id": "1"})
	require.False(t, res.IsError, text(t, res))
	var contact map[string]any
	require.NoError(t, json.Unmarshal([]byte(text(t, res)), &contact))
	assert.Equal(t, "alice@example.com", contact["email"], "fixture contact 1")

	// Query parameters.
	res = call(t, s, "contacts_list", map[string]any{"pageSize": 2})
	require.False(t, res.IsError, text(t, res))
	var page struct {
		Items    []map[string]any `json:"items"`
		PageSize int              `json:"pageSize"`
	}
	require.NoError(t, json.Unmarshal([]byte(text(t, res)), &page))
	assert.Equal(t, 2, page.PageSize)
	assert.Len(t, page.Items, 2)

	// Body fields are top-level arguments.
	res = call(t, s, "contacts_create", map[string]any{"email": "mcp-new@example.com", "firstName": "Mcp"})
	require.False(t, res.IsError, text(t, res))
	var created map[string]any
	require.NoError(t, json.Unmarshal([]byte(text(t, res)), &created))
	assert.Equal(t, "mcp-new@example.com", created["email"])

	// Path parameter and body together; 204 carries no body.
	res = call(t, s, "contacts_update", map[string]any{"id": created["id"], "firstName": "Renamed"})
	require.False(t, res.IsError, text(t, res))
	res = call(t, s, "contacts_delete", map[string]any{"id": created["id"]})
	require.False(t, res.IsError, text(t, res))
}

// Tags are tools: a name with a slash travels as a path parameter, and the contact id
// as another.
func TestMCPTagToolsApplyAndRemove(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.MCPClient(t, seedToken(t, env, 1, []string{"contacts:read", "contacts:write"}))

	res := call(t, s, "tags_apply", map[string]any{"contactId": "2", "name": "plan / pro"})
	require.False(t, res.IsError, text(t, res))
	assert.Contains(t, text(t, res), "plan / pro")

	res = call(t, s, "tags_list_for_contact", map[string]any{"contactId": "2"})
	require.False(t, res.IsError, text(t, res))
	assert.Contains(t, text(t, res), "plan / pro")

	res = call(t, s, "tags_remove", map[string]any{"contactId": "2", "name": "plan / pro"})
	require.False(t, res.IsError, text(t, res))

	res = call(t, s, "tags_list_for_contact", map[string]any{"contactId": "2"})
	require.False(t, res.IsError, text(t, res))
	assert.NotContains(t, text(t, res), "plan / pro")
}

func TestMCPToolCallReturnsTheAPIError(t *testing.T) {
	env := testhelper.Setup(t)
	ro := env.MCPClient(t, seedToken(t, env, 1, []string{"contacts:read"}))

	// Scope: the /api insufficient-scope problem comes back as a tool error.
	res := call(t, ro, "contacts_create", map[string]any{"email": "nope@example.com"})
	require.True(t, res.IsError)
	assert.Contains(t, text(t, res), "insufficient scope")

	// Not found.
	res = call(t, ro, "contacts_get", map[string]any{"id": "999999"})
	require.True(t, res.IsError)
	assert.Contains(t, text(t, res), "404")
}

func TestMCPCallsAreIsolatedToTheTokensWorkspace(t *testing.T) {
	env := testhelper.Setup(t)
	other, err := env.DB.Workspace.Create().SetName("Other").SetSlug("other").
		SetCollectKey("omck_other").SetIngestKey("omik_other").Save(context.Background())
	require.NoError(t, err)
	s := env.MCPClient(t, seedToken(t, env, int(other.ID), []string{"contacts:read"}))

	res := call(t, s, "contacts_get", map[string]any{"id": "1"})
	require.True(t, res.IsError, "contact 1 belongs to workspace acme")

	res = call(t, s, "contacts_list", nil)
	require.False(t, res.IsError, text(t, res))
	assert.NotContains(t, text(t, res), "alice@example.com")
}

// Consent only narrows through the agent (ADR 0016): MCP can add a Suppression or
// record an Unsubscribe, and exposes nothing that resubscribes or lifts one.
func TestMCPConsentOnlyNarrows(t *testing.T) {
	env := testhelper.Setup(t)
	s := env.MCPClient(t, seedToken(t, env, 1, []string{"contacts:write"}))

	list, err := s.ListTools(context.Background(), nil)
	require.NoError(t, err)
	for _, tool := range list.Tools {
		name := strings.ToLower(tool.Name)
		assert.NotContains(t, name, "resubscribe")
		if strings.Contains(name, "suppression") || strings.Contains(name, "unsubscribe") {
			assert.True(t, strings.HasSuffix(name, "_create"), "%s: consent tools only create", tool.Name)
			require.NotNil(t, tool.Annotations)
			require.NotNil(t, tool.Annotations.DestructiveHint)
			assert.False(t, *tool.Annotations.DestructiveHint)
		}
	}

	res := call(t, s, "unsubscribes_create", map[string]any{"destination": "alice@example.com"})
	require.False(t, res.IsError, text(t, res))
	assert.Contains(t, text(t, res), `"sendingSource":"broadcasts"`)

	res = call(t, s, "suppressions_create", map[string]any{"destination": "alice@example.com"})
	require.False(t, res.IsError, text(t, res))
	assert.Contains(t, text(t, res), `"reason":"manual"`)
}
