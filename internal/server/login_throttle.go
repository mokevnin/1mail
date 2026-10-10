package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"net/url"

	"github.com/go-chi/httprate"

	"github.com/mokevnin/1mail/internal/accounts"
	"github.com/mokevnin/1mail/internal/clientip"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/ratelimit"
)

// loginThrottle wraps the go-pkgz/auth login route (ADR 0024). The provider maps a
// credential-checker error to 500 and a wrong password to a fixed 403, so it cannot
// answer 429 itself; this wrapper does, before the provider runs: first the per-IP
// cap, then the per-account delay, which applies to a correct password too (else the
// delay could be bypassed by guessing until one attempt succeeds). The credential
// checker only records successes and failures.
func loginThrottle(next http.Handler, attempts *accounts.Attempts, perIP *ratelimit.Policy) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !perIP.Allow(w, r, "login|"+httprate.CanonicalizeIP(clientip.FromContext(r.Context()))) {
			return
		}
		if email := loginEmail(r); email != "" {
			wait, err := attempts.Delay(r.Context(), accounts.KindLogin, email)
			if err != nil {
				logging.FromContext(r.Context()).Error("login attempt lookup failed", "err", err)
				writeProblem(w, http.StatusInternalServerError, "internal server error")
				return
			}
			if wait > 0 {
				ratelimit.Wait(w, r, ratelimit.PolicyLoginAccount, attempts.Limit(accounts.KindLogin), wait, attempts.Now())
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// loginEmail reads the address the provider will check, the way the provider reads
// it (query on GET; JSON or form body on POST), and puts the body back so the
// provider still sees it. "" means no address could be read: the provider answers
// 400 and nothing is checked, so there is nothing to throttle.
func loginEmail(r *http.Request) string {
	switch r.Method {
	case http.MethodGet:
		return r.URL.Query().Get("user")
	case http.MethodPost:
	default:
		return ""
	}
	if r.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(r.Body)
	// Hand the provider exactly what was read, even on a read error (an oversized
	// body surfaces its own error there).
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(raw), errReader{err}))
	if err != nil {
		return ""
	}
	if mt, _, perr := mime.ParseMediaType(r.Header.Get("Content-Type")); perr == nil && mt == "application/json" {
		var creds struct {
			User string `json:"user"`
		}
		if json.Unmarshal(raw, &creds) != nil {
			return ""
		}
		return creds.User
	}
	form, perr := url.ParseQuery(string(raw))
	if perr != nil {
		return ""
	}
	if user := form.Get("user"); user != "" {
		return user
	}
	return r.URL.Query().Get("user")
}

// errReader replays the error that ended a read (nil reads as end of body).
type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) {
	if e.err != nil {
		return 0, e.err
	}
	return 0, io.EOF
}
