// Package ratelimit is the in-binary request limiter (ADR 0018). Flood limits are
// in-memory sliding windows (go-chi/httprate); a limit of 0 disables its policy.
//
// Seams for the other slices:
//   - NewPolicy builds one named limiter; Policy.Allow checks a key and, on a
//     rejection, answers the 429 itself (headers, problem body, metric, log);
//   - Rejected is the policy-named metric + warn log for callers that render the
//     429 their own way (e.g. a typed error from an ogen security handler);
//   - WriteProblem renders the 429 body.
package ratelimit

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/httprate"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/internal/clientip"
	"github.com/mokevnin/1mail/internal/logging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// Policy names. They are the `policy` label of ratelimit_rejected_total, so they
// must stay a small bounded set.
const (
	PolicyHuman = "human"
	// PolicyLoginAccount is the per-account login delay, PolicyLoginIP the per-IP cap
	// on login requests.
	PolicyLoginAccount = "login-account"
	PolicyLoginIP      = "login-ip"
)

const window = time.Minute

// Policy is one named limiter. The zero value and nil allow everything, which is
// what a disabled (0) limit builds.
type Policy struct {
	name string
	rl   *httprate.RateLimiter
}

// NewPolicy builds a limiter of limit requests per window. It returns nil when
// limit is not positive (disabled).
func NewPolicy(name string, limit int, window time.Duration) *Policy {
	if limit <= 0 {
		return nil
	}
	return &Policy{name: name, rl: httprate.NewRateLimiter(limit, window)}
}

// Allow counts one request against key and reports whether it may proceed. On a
// rejection it has already answered 429 (X-RateLimit-* and Retry-After headers,
// problem body), counted and logged it, and the caller must stop.
func (p *Policy) Allow(w http.ResponseWriter, r *http.Request, key string) bool {
	if p == nil || !p.rl.OnLimit(w, r, key) {
		return true
	}
	Rejected(r.Context(), p.name)
	WriteProblem(w)
	return false
}

// Rejected records one rejection under policy: ratelimit_rejected_total{policy} and
// a warn log. It carries the policy and the client address only, never an email or
// a path (paths hold tokens).
func Rejected(ctx context.Context, policy string) {
	counter, err := otel.Meter("1mail/ratelimit").Int64Counter("ratelimit_rejected",
		metric.WithDescription("Requests refused by a rate limit, by policy."))
	if err == nil {
		counter.Add(ctx, 1, metric.WithAttributes(attribute.String("policy", policy)))
	}
	logging.FromContext(ctx).Warn("rate limit exceeded", "policy", policy, "ip", clientip.FromContext(ctx))
}

// WriteProblem renders the RFC 7807 429 body. Headers already on w (Retry-After,
// X-RateLimit-*, CORS) are kept; the Retry-After seconds are repeated in the body
// as retryAfter, so a client that only sees the body can show the wait.
func WriteProblem(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusTooManyRequests)
	body := map[string]any{
		"status": http.StatusTooManyRequests,
		"title":  http.StatusText(http.StatusTooManyRequests),
		"detail": "rate limit exceeded, retry later",
	}
	if secs, err := strconv.Atoi(w.Header().Get("Retry-After")); err == nil {
		body["retryAfter"] = secs
	}
	_ = json.NewEncoder(w).Encode(body)
}

// Wait answers 429 for a limit that is a delay rather than a window budget (the
// per-account login delay): the caller must wait `wait` from now. limit is the
// budget reported in X-RateLimit-Limit; Remaining is always 0. It counts and logs the
// rejection under policy.
func Wait(w http.ResponseWriter, r *http.Request, policy string, limit int, wait time.Duration, now time.Time) {
	secs := int(math.Ceil(wait.Seconds()))
	h := w.Header()
	h.Set("Retry-After", strconv.Itoa(secs))
	h.Set("X-RateLimit-Limit", strconv.Itoa(limit))
	h.Set("X-RateLimit-Remaining", "0")
	h.Set("X-RateLimit-Reset", strconv.FormatInt(now.Add(wait).Unix(), 10))
	Rejected(r.Context(), policy)
	WriteProblem(w)
}

// Limiter holds every policy built from the configured limits.
type Limiter struct {
	human   *Policy
	loginIP *Policy
}

// New builds the policies from limits.
func New(limits config.RateLimits) *Limiter {
	return &Limiter{
		human:   NewPolicy(PolicyHuman, limits.Human, window),
		loginIP: NewPolicy(PolicyLoginIP, limits.LoginIP, window),
	}
}

// LoginIP is the per-IP login limiter (nil when disabled). The login route's wrapper
// applies it, because that wrapper owns the route.
func (l *Limiter) LoginIP() *Policy { return l.loginIP }

// Middleware applies the policy of the request's route. It sits after the client
// address middleware (the key is clientip, never a raw header) and before timeout.
// Operational endpoints are never limited, whatever policies exist.
func (l *Limiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !exempt(r.URL.Path) {
			if route := humanRoute(r); route != "" &&
				!l.human.Allow(w, r, httprate.CanonicalizeIP(clientip.FromContext(r.Context()))+"|"+route) {
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// exempt lists what is never limited: probes, metrics and the provider hooks
// (authenticated by their ingest key).
func exempt(path string) bool {
	return path == "/healthz" || path == "/readyz" || path == "/metrics" || strings.HasPrefix(path, "/hooks/")
}

// humanRoute names the public human-facing endpoints (one bucket per IP and
// route): signup, invitation accept and consent confirm. "" means not one.
func humanRoute(r *http.Request) string {
	if r.Method != http.MethodPost {
		return ""
	}
	p := r.URL.Path
	switch {
	case p == "/site/auth/register":
		return "signup"
	case strings.HasPrefix(p, "/site/invitations/") && strings.HasSuffix(p, "/accept"):
		return "invitation-accept"
	case strings.HasPrefix(p, "/site/confirmations/") && !strings.Contains(strings.TrimPrefix(p, "/site/confirmations/"), "/"):
		return "consent-confirm"
	}
	return ""
}
