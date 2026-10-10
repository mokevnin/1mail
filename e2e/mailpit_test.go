//go:build e2e

package e2e

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeInbox answers Mailpit's search with nothing and its list with a canned inbox,
// the shape of a mail that went to the wrong address.
func fakeInbox(t *testing.T) *Mailpit {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/search", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[]}`))
	})
	mux.HandleFunc("/api/v1/messages", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"messages":[{"ID":"1","From":{"Address":"news@a.test"},"To":[{"Address":"other@b.test"}],"Subject":"Wrong place"}]}`))
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mp := NewMailpit(srv.URL)
	mp.poll = 10 * time.Millisecond
	return mp
}

func TestWaitForRecipientTimesOutWithAnInboxListing(t *testing.T) {
	t.Parallel()
	_, err := fakeInbox(t).WaitForRecipient(t.Context(), "want@b.test", 150*time.Millisecond)

	require.Error(t, err)
	assert.True(t, IsNoMail(err))
	assert.ErrorContains(t, err, "want@b.test")
	assert.ErrorContains(t, err, "other@b.test")
	assert.ErrorContains(t, err, "Wrong place")
}

func TestWaitForRecipientReportsAnEmptyInbox(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte(`{"messages":[]}`)) })
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	mp := NewMailpit(srv.URL)
	mp.poll = 10 * time.Millisecond

	_, err := mp.WaitForRecipient(t.Context(), "want@b.test", 100*time.Millisecond)
	assert.ErrorContains(t, err, "the inbox is empty")
}
