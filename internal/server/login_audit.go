package server

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/url"
	"strings"

	"github.com/mokevnin/1mail/internal/accounts"
)

// loginBodyLimit caps how much of a login body is read to learn who is signing in.
const loginBodyLimit = 64 << 10

// auditLogin wraps the go-pkgz/auth direct login handler: when it answers 200 (the
// credentials were accepted and the JWT cookie issued) the sign-in is recorded as
// `user.login` in the log of every Workspace the User belongs to (ADR 0022). A
// failed login is not recorded. The wrapped provider knows nothing of requests or
// Workspaces, so the recording happens here, with the request's own context (request
// id, address, user agent).
func auditLogin(next http.Handler, acc *accounts.Accounts) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		login := peekLogin(r)
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		if rec.status != http.StatusOK || login == "" {
			return
		}
		u, err := acc.UserByEmail(r.Context(), login)
		if err == nil {
			err = acc.RecordLogin(r.Context(), u)
		}
		if err != nil {
			// The sign-in already happened; the entry cannot veto it.
			slog.ErrorContext(r.Context(), "audit: login not recorded", "error", err)
		}
	})
}

// peekLogin reads the login (email) from a JSON or form login body and restores the
// body for the wrapped handler.
func peekLogin(r *http.Request) string {
	if r.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, loginBodyLimit))
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), r.Body))
	if err != nil {
		return ""
	}
	if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt == "application/json" {
		var in struct {
			User string `json:"user"`
		}
		if json.Unmarshal(raw, &in) != nil {
			return ""
		}
		return strings.TrimSpace(in.User)
	}
	form, err := url.ParseQuery(string(raw))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(form.Get("user"))
}

// statusRecorder remembers the status code the wrapped handler wrote.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Unwrap() http.ResponseWriter { return s.ResponseWriter }
