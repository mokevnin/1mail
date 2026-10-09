package server

import (
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func do(t *testing.T, h http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestProbesOnlyAnswerGet(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://127.0.0.1:1/none")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	for name, h := range map[string]http.Handler{"healthz": healthzHandler(), "readyz": readyzHandler(db)} {
		rec := do(t, h, http.MethodPost, "/"+name, nil)
		assert.Equal(t, http.StatusMethodNotAllowed, rec.Code, name)
		assert.Equal(t, http.MethodGet, rec.Header().Get("Allow"), name)
	}
}

func TestReadyzIsUnavailableWhenTheDatabaseIsUnreachable(t *testing.T) {
	db, err := sql.Open("pgx", "postgres://127.0.0.1:1/none?connect_timeout=1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	rec := do(t, readyzHandler(db), http.MethodGet, "/readyz", nil)
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestProblemErrorHandlerRendersOgenErrorsAsProblemJSON(t *testing.T) {
	// A client error keeps its status and explains itself.
	rec := httptest.NewRecorder()
	problemErrorHandler(t.Context(), rec, nil, &ogenerrors.DecodeRequestError{Err: errors.New("bad body")})
	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
	assert.Contains(t, rec.Body.String(), `"title":"Bad Request"`)
	assert.Contains(t, rec.Body.String(), "decode request: bad body")

	// Anything else is a 500 that never leaks the cause.
	rec = httptest.NewRecorder()
	problemErrorHandler(t.Context(), rec, nil, errors.New("db password is hunter2"))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.JSONEq(t, `{"status":500,"title":"Internal Server Error"}`, rec.Body.String())
}

func TestRecovererTurnsAPanicIntoA500(t *testing.T) {
	h := recoverer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := do(t, h, http.MethodGet, "/x", nil)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
}

func TestRequestIDIsEchoedOrGenerated(t *testing.T) {
	h := requestID(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	assert.Equal(t, "abc-123", do(t, h, http.MethodGet, "/", map[string]string{"X-Request-Id": "abc-123"}).Header().Get("X-Request-Id"))
	generated := do(t, h, http.MethodGet, "/", nil).Header().Get("X-Request-Id")
	assert.Len(t, generated, 26)
	assert.NotEqual(t, generated, do(t, h, http.MethodGet, "/", nil).Header().Get("X-Request-Id"))
}

func TestTimeoutBoundsTheRequestContext(t *testing.T) {
	var deadline time.Time
	var ok bool
	h := timeout(time.Minute)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		deadline, ok = r.Context().Deadline()
	}))
	do(t, h, http.MethodGet, "/", nil)
	require.True(t, ok)
	assert.WithinDuration(t, time.Now().Add(time.Minute), deadline, 5*time.Second)
}

func TestCORSPoliciesByPath(t *testing.T) {
	ok := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	preflight := func(h http.Handler, path string) http.Header {
		return do(t, h, http.MethodOptions, path, map[string]string{
			"Origin":                        "https://customer.example",
			"Access-Control-Request-Method": "POST",
		}).Header()
	}

	// The collect API echoes any origin but never allows credentials.
	for _, origins := range [][]string{nil, {"https://app.example"}} {
		h := corsMiddleware(origins)(ok)
		got := preflight(h, "/collect/v1/track")
		assert.Equal(t, "https://customer.example", got.Get("Access-Control-Allow-Origin"))
		assert.Empty(t, got.Get("Access-Control-Allow-Credentials"))
	}

	// The app reflects any origin when no allowlist is set, with credentials...
	open := preflight(corsMiddleware(nil)(ok), "/site/workspaces")
	assert.Equal(t, "https://customer.example", open.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", open.Get("Access-Control-Allow-Credentials"))

	// ...and enforces the allowlist when one is configured.
	listed := corsMiddleware([]string{"https://app.example"})(ok)
	assert.Empty(t, preflight(listed, "/site/workspaces").Get("Access-Control-Allow-Origin"))
	allowed := do(t, listed, http.MethodOptions, "/site/workspaces", map[string]string{
		"Origin": "https://app.example", "Access-Control-Request-Method": "POST",
	}).Header()
	assert.Equal(t, "https://app.example", allowed.Get("Access-Control-Allow-Origin"))
}

func TestChainRunsMiddlewareOutermostFirst(t *testing.T) {
	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	do(t, chain(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { order = append(order, "handler") }), mw("a"), mw("b")), http.MethodGet, "/", nil)
	assert.Equal(t, []string{"a", "b", "handler"}, order)
}
