package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

func TestHealthz(t *testing.T) {
	env := testhelper.Setup(t)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	env.Server.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}

func TestReadyz(t *testing.T) {
	env := testhelper.Setup(t)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()
	env.Server.ServeHTTP(w, req)

	// The txdb test connection is live, so readiness reports OK.
	assert.Equal(t, http.StatusOK, w.Code)
	assert.JSONEq(t, `{"status":"ok"}`, w.Body.String())
}
