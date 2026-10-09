package server_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mokevnin/1mail/internal/fixtures"
	"github.com/mokevnin/1mail/internal/testhelper"
	"github.com/stretchr/testify/assert"
)

// An unknown ingest key is rejected (404) before any signature work — the
// workspace is resolved from the key first.
func TestSESHookUnknownKeyNotFound(t *testing.T) {
	env := testhelper.Setup(t)

	req := httptest.NewRequestWithContext(t.Context(), "POST", "/hooks/omik_does_not_exist/ses",
		strings.NewReader(`{"Type":"Notification","Message":"{}"}`))
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, 404, rec.Code)
}

// A known workspace key with an unsigned payload is rejected by SNS signature
// verification (403), confirming the verify gate is wired ahead of processing.
func TestSESHookRejectsUnverifiedPayload(t *testing.T) {
	env := testhelper.Setup(t)

	// The Acme workspace ingest key resolves the workspace before verification.
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/hooks/"+fixtures.AcmeIngestKey+"/ses",
		strings.NewReader(`{"Type":"Notification","Message":"{}","Signature":"bogus","SigningCertURL":"https://sns.us-east-1.amazonaws.com/x.pem"}`))
	rec := httptest.NewRecorder()
	env.Server.ServeHTTP(rec, req)

	assert.Equal(t, 403, rec.Code)
}
