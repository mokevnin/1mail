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
	"net/http"
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
// X-RateLimit-*, CORS) are kept.
func WriteProblem(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": http.StatusTooManyRequests,
		"title":  http.StatusText(http.StatusTooManyRequests),
		"detail": "rate limit exceeded, retry later",
	})
}

// Limiter holds every policy built from the configured limits.
type Limiter struct {
	human *Policy
}

// New builds the policies from limits.
func New(limits config.RateLimits) *Limiter {
	return &Limiter{human: NewPolicy(PolicyHuman, limits.Human, window)}
}

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
