//go:build e2e

package e2e

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeMail is a canned row of the fake Mailpit inbox.
type fakeMail struct {
	id, to, subject string
}

// fakeMailpit serves the Mailpit endpoints the Inbox uses from a canned inbox and
// records the ids it was asked to delete.
type fakeMailpit struct {
	mu      sync.Mutex
	mails   []fakeMail
	deleted []string
}

func (f *fakeMailpit) add(m fakeMail) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.mails = append(f.mails, m)
}

func (f *fakeMailpit) deletedIDs() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.deleted...)
}

func (f *fakeMailpit) rows(substr string) []map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	rows := []map[string]any{}
	for _, m := range f.mails {
		if strings.Contains(strings.ToLower(m.to), strings.ToLower(substr)) {
			rows = append(rows, map[string]any{
				"ID": m.id, "From": map[string]string{"Address": "news@a.test"},
				"To": []map[string]string{{"Address": m.to}}, "Subject": m.subject,
			})
		}
	}
	return rows
}

// newFakeInbox starts the fake server and returns an Inbox over it.
func newFakeInbox(t *testing.T) (*Inbox, *fakeMailpit) {
	t.Helper()
	f := &fakeMailpit{}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/search", func(w http.ResponseWriter, r *http.Request) {
		q := strings.Trim(strings.TrimPrefix(r.URL.Query().Get("query"), "to:"), `"`)
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": f.rows(q)})
	})
	mux.HandleFunc("GET /api/v1/messages", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"messages": f.rows("")})
	})
	mux.HandleFunc("DELETE /api/v1/messages", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ IDs []string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		f.mu.Lock()
		f.deleted = append(f.deleted, body.IDs...)
		f.mu.Unlock()
		_, _ = w.Write([]byte(`"ok"`))
	})
	mux.HandleFunc("GET /api/v1/message/{id}/headers", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	})
	mux.HandleFunc("GET /api/v1/message/{id}", func(w http.ResponseWriter, r *http.Request) {
		for _, row := range f.rows("") {
			if row["ID"] == r.PathValue("id") {
				_ = json.NewEncoder(w).Encode(row)
				return
			}
		}
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	mp := NewMailpit(srv.URL)
	mp.poll = 10 * time.Millisecond
	return &Inbox{t: t, mp: mp}, f
}

func TestWaitTimesOutWithAnInboxListing(t *testing.T) {
	t.Parallel()
	in, f := newFakeInbox(t)
	f.add(fakeMail{id: "1", to: "other@b.test", subject: "Wrong place"})

	_, err := in.wait(t.Context(), Match{To: "want@b.test"}, 150*time.Millisecond)

	require.Error(t, err)
	assert.ErrorContains(t, err, "want@b.test")
	assert.ErrorContains(t, err, "other@b.test")
	assert.ErrorContains(t, err, "Wrong place")
}

func TestWaitReportsAnEmptyInbox(t *testing.T) {
	t.Parallel()
	in, _ := newFakeInbox(t)

	_, err := in.wait(t.Context(), Match{To: "want@b.test"}, 100*time.Millisecond)

	assert.ErrorContains(t, err, "the inbox is empty")
}

func TestWaitMatchesTheRecipientExactlyAndCaseInsensitively(t *testing.T) {
	t.Parallel()
	in, f := newFakeInbox(t)
	f.add(fakeMail{id: "1", to: "xwant@b.test", subject: "Superstring"})
	f.add(fakeMail{id: "2", to: "Want@B.test", subject: "Right"})

	msg, err := in.wait(t.Context(), Match{To: "want@b.test"}, time.Second)

	require.NoError(t, err)
	assert.Equal(t, "Right", msg.Subject)
}

func TestWaitMatchesTheSubjectExactly(t *testing.T) {
	t.Parallel()
	in, f := newFakeInbox(t)
	f.add(fakeMail{id: "1", to: "want@b.test", subject: "First issue"})
	f.add(fakeMail{id: "2", to: "want@b.test", subject: "Second"})

	msg, err := in.wait(t.Context(), Match{To: "want@b.test", Subject: "Second"}, time.Second)
	require.NoError(t, err)
	assert.Equal(t, "2", msg.ID)

	_, err = in.wait(t.Context(), Match{To: "want@b.test", Subject: "Second issue"}, 100*time.Millisecond)
	assert.Error(t, err, "a different subject is not a match")
}

func TestRequireNoneReturnsWhenNothingMatches(t *testing.T) {
	t.Parallel()
	in, f := newFakeInbox(t)
	f.add(fakeMail{id: "1", to: "want@b.test", subject: "Other subject"})

	in.RequireNone(Match{To: "want@b.test", Subject: "Absent"}, 150*time.Millisecond)
}

func TestInboxDeletesEveryObservedMessageOnCleanup(t *testing.T) {
	t.Parallel()
	in, f := newFakeInbox(t)
	f.add(fakeMail{id: "1", to: "a@b.test", subject: "waited"})
	f.add(fakeMail{id: "2", to: "c@b.test", subject: "seen by absence only"})
	f.add(fakeMail{id: "3", to: "d@b.test", subject: "untouched"})

	_, err := in.wait(t.Context(), Match{To: "a@b.test"}, time.Second)
	require.NoError(t, err)
	// An absence check that finds a match records it too (the check itself then fails).
	assert.True(t, in.seen(Match{To: "c@b.test"}))

	in.cleanup()

	assert.ElementsMatch(t, []string{"1", "2"}, f.deletedIDs())
}
