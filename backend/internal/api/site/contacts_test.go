package site_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/mokevnin/1mail/backend/internal/testhelper"
)

func TestContactsList(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	req := httptest.NewRequest(http.MethodGet, "/site/contacts", nil)
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body struct {
		Items      []any `json:"items"`
		TotalItems int   `json:"totalItems"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, 3, body.TotalItems)
	assert.Len(t, body.Items, 3)
}

func TestContactsCreate(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	body := `{"email":"new@example.com","firstName":"New"}`
	req := httptest.NewRequest(http.MethodPost, "/site/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusCreated, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "new@example.com", resp["email"])
}

func TestContactsCreate_Conflict(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	body := `{"email":"alice@example.com"}`
	req := httptest.NewRequest(http.MethodPost, "/site/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusConflict, rec.Code)
}

func TestContactsGet(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	req := httptest.NewRequest(http.MethodGet, "/site/contacts/1", nil)
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "alice@example.com", resp["email"])
}

func TestContactsGet_NotFound(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	req := httptest.NewRequest(http.MethodGet, "/site/contacts/9999", nil)
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestContactsDelete(t *testing.T) {
	env := testhelper.Setup(t)
	env.LoadFixtures(t)

	req := httptest.NewRequest(http.MethodDelete, "/site/contacts/1", nil)
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNoContent, rec.Code)
}
