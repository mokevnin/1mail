package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"runtime/debug"
	"strings"
	"time"

	goauth "github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	apiauth "github.com/mokevnin/1mail/internal/api/auth"
	apicollect "github.com/mokevnin/1mail/internal/api/collect"
	apiexternal "github.com/mokevnin/1mail/internal/api/external"
	apisite "github.com/mokevnin/1mail/internal/api/site"
	"github.com/mokevnin/1mail/internal/clientip"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/oauthserver"
	"github.com/mokevnin/1mail/internal/telemetry"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/oklog/ulid/v2"
	"github.com/rs/cors"
)

// New builds the top-level net/http handler wiring the three ogen-generated
// API servers (site, external, collect) plus go-pkgz/auth endpoints.
// New composes the HTTP handler. client is the raw ent client: only the pieces whose
// Workspace is not known up front take it (auth, OAuth, tracking, provider hooks);
// the site, external and collect handlers get none (ADR 0017).
func New(cfg *config.Config, db *sql.DB, client *ent.Client, site apisite.Deps, external, mcp http.Handler) (http.Handler, error) {
	bus := site.Bus
	mux := http.NewServeMux()

	// Send the JWT cookie with the Secure attribute whenever the instance is served
	// over HTTPS (AppURL reflects the public scheme, so this holds even though Caddy
	// terminates TLS and the Go server itself listens on plain HTTP). Stays false for
	// local http://localhost dev, where a Secure cookie would never be sent back.
	secureCookies := strings.HasPrefix(cfg.AppURL, "https://")

	// Auth service (go-pkgz/auth) — JWT issuance + direct (email/password) provider.
	authSvc := goauth.NewService(goauth.Opts{
		SecretReader:   token.SecretFunc(func(string) (string, error) { return cfg.JWTSecret, nil }),
		TokenDuration:  time.Hour,
		CookieDuration: 24 * time.Hour,
		// Cross-site writes are rejected by crossOriginGuard instead (see there).
		DisableXSRF:    true,
		SameSiteCookie: http.SameSiteLaxMode,
		SecureCookies:  secureCookies,
		Issuer:         "1mail",
		URL:            cfg.AppURL,
		AvatarStore:    avatar.NewLocalFS("/tmp/1mail-avatars"),
	})
	authSvc.AddDirectProvider("direct", apiauth.NewCredChecker(client))
	authHandler, avatarHandler := authSvc.Handlers()
	mux.Handle("/auth/", authHandler)
	mux.Handle("/avatar/", avatarHandler)
	// The SPA's generated client posts to /site/auth/direct/login (baseUrl "/site");
	// route that exact path to the go-pkgz/auth direct provider, which issues the JWT
	// cookie. go-pkgz/auth routes by path suffix, so the /site prefix is harmless, and
	// the exact pattern outranks the /site/ subtree below without shadowing /site/auth/register.
	mux.Handle("/site/auth/direct/login", authHandler)

	// Site API — /site (JWT cookie via generated SecurityHandler; register and
	// direct-login are public per the spec).
	siteSrv, err := siteapi.NewServer(
		apisite.NewHandlers(site),
		apiauth.NewSiteSecurityHandler(cfg.JWTSecret, client),
		siteapi.WithPathPrefix("/site"),
		siteapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/site/", siteSrv)

	// External API — /api (Bearer token auth via ogen SecurityHandler); built by
	// NewExternalAPI so the MCP surface dispatches through the same server.
	mux.Handle("/api/", external)

	// MCP — /mcp (ADR 0016): tools projected from the external contract and
	// dispatched in-process through the handler above with the caller's own token.
	mux.Handle("/mcp", mcp)

	// OAuth 2.1 for MCP connectors (ADR 0016): discovery, dynamic client
	// registration, authorize (hands off to the SPA consent screen) and token.
	oauthserver.New(client, cfg.AppURL).Mount(mux)

	// Collect API — /collect (x-collect-key via generated SecurityHandler).
	colSrv, err := collectapi.NewServer(
		apicollect.NewHandlers(bus),
		apiauth.NewCollectSecurityHandler(client),
		collectapi.WithPathPrefix("/collect"),
		collectapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/collect/", colSrv)

	// Liveness/readiness probes (no auth) for orchestrators and load balancers.
	mux.Handle("/healthz", healthzHandler())
	mux.Handle("/readyz", readyzHandler(db))

	// Prometheus metrics exposition (no auth, like the probes). Operators should
	// restrict scrape access at the ingress edge. A 503 stub until telemetry.Setup
	// runs, so the mount is safe even when telemetry is disabled (e.g. tests).
	mux.Handle("/metrics", telemetry.MetricsHandler())

	// Public tracking snippet (no auth) — embedded IIFE bundle.
	mux.Handle("/t.js", trackerHandler())

	// Public email engagement endpoints (open pixel, click redirect, unsubscribe).
	mux.Handle("/e/", trackingHandler(client, bus, site.Tracker))

	// Inbound provider webhooks (SES bounce/complaint via SNS), routed by the
	// workspace's secret ingest key: POST /hooks/{key}/{provider}.
	mux.Handle("/hooks/", hooksHandler(client, bus))

	// Catch-all: the embedded SPA (release builds with -tags embed_spa). Most
	// specific pattern wins, so this never shadows the API prefixes above.
	mux.Handle("/", spaHandler(cfg.Locale))

	guard, err := crossOriginGuard(cfg.AppURL, cfg.CORSOrigins)
	if err != nil {
		return nil, err
	}

	// requestID is outermost so the correlation id is in context before recoverer
	// runs — the panic log then carries request_id. (requestID is trivial and
	// cannot itself panic, so nothing downstream of recovery is lost.)
	// guard sits inside corsMiddleware so preflights are answered before the check.
	return chain(mux, requestID, clientip.Middleware, recoverer, timeout(30*time.Second), bodyLimit(cfg.MaxBodyBytes, cfg.CollectMaxBodyBytes), corsMiddleware(cfg.CORSOrigins), guard), nil
}

// NewExternalAPI builds the external API (/api) ogen server: Bearer API-token
// auth, RFC 7807 errors, mounted under the /api prefix.
func NewExternalAPI(client *ent.Client, deps apiexternal.Deps) (http.Handler, error) {
	return externalapi.NewServer(
		apiexternal.NewHandlers(deps),
		apiauth.NewExternalSecurityHandler(client),
		externalapi.WithPathPrefix("/api"),
		externalapi.WithErrorHandler(problemErrorHandler),
	)
}

// problemErrorHandler renders ogen errors as RFC 7807 application/problem+json.
// A reference id from another Workspace (the scoped client's ErrNotInWorkspace,
// ADR 0017) is a client error, so it is a 422 and never a 500.
func problemErrorHandler(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	code := http.StatusInternalServerError
	var oe ogenerrors.Error
	var tooBig *http.MaxBytesError
	switch {
	case errors.As(err, &tooBig):
		code = http.StatusRequestEntityTooLarge
	case errors.As(err, &oe):
		code = oe.Code()
	case errors.Is(err, ent.ErrNotInWorkspace):
		code = http.StatusUnprocessableEntity
	}
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	prob := map[string]any{
		"status": code,
		"title":  http.StatusText(code),
	}
	if code < http.StatusInternalServerError {
		prob["detail"] = err.Error()
	}
	_ = json.NewEncoder(w).Encode(prob)
}

func writeProblem(w http.ResponseWriter, code int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status": code,
		"title":  http.StatusText(code),
		"detail": detail,
	})
}

// --- cross-cutting net/http middleware ---

func chain(h http.Handler, mws ...func(http.Handler) http.Handler) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

func recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := logging.FromContext(r.Context())
		defer func() {
			if rec := recover(); rec != nil {
				logger.Error("panic recovered",
					"err", rec,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				writeProblem(w, http.StatusInternalServerError, "internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// timeout bounds every request's context (hang protection; replaces echo ContextTimeout).
func timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bodyLimit caps every request body before a handler or the ogen decoder reads it:
// collect gets its own (smaller) cap, everything else the default. An oversized
// body surfaces as *http.MaxBytesError, which problemErrorHandler renders as 413.
func bodyLimit(def, collect int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := def
			if strings.HasPrefix(r.URL.Path, "/collect/") {
				limit = collect
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = ulid.Make().String()
		}
		w.Header().Set("X-Request-Id", id)
		// Carry the id in context so request-scoped logs (via logging.FromContext)
		// correlate back to this request.
		next.ServeHTTP(w, r.WithContext(logging.WithRequestID(r.Context(), id)))
	})
}

// cookiePath reports whether a path is authenticated by the JWT cookie (the SPA
// API and go-pkgz/auth), the only surface exposed to CSRF.
func cookiePath(path string) bool {
	return strings.HasPrefix(path, "/site/") || strings.HasPrefix(path, "/auth/")
}

// corsMiddleware applies three rs/cors policies by path:
//   - /collect/: echoes any origin without credentials (the collect key is
//     public, cookies are first-party on the customer's own domain);
//   - /site/ and /auth/ (cookie auth): credentials only for the configured
//     allowlist. The SPA is served same-origin, so with no allowlist there are no
//     CORS headers at all;
//   - everything else (external API, MCP, OAuth: bearer tokens): echoes any origin
//     without credentials.
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions}
	// "*" reflects whatever request headers the client asks for (works with
	// credentials, and avoids brittle exact-match of comma-joined header lists).
	anyHeaders := []string{"*"}
	open := cors.New(cors.Options{
		AllowedMethods:  methods,
		AllowedHeaders:  anyHeaders,
		AllowOriginFunc: func(string) bool { return true },
	})
	collect := cors.New(cors.Options{
		AllowedMethods:  []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:  anyHeaders,
		AllowOriginFunc: func(string) bool { return true },
	})
	var cookie *cors.Cors
	if len(origins) > 0 {
		cookie = cors.New(cors.Options{
			AllowedMethods:   methods,
			AllowedHeaders:   anyHeaders,
			AllowCredentials: true,
			AllowedOrigins:   origins,
		})
	}

	return func(next http.Handler) http.Handler {
		openH := open.Handler(next)
		collectH := collect.Handler(next)
		cookieH := next
		if cookie != nil {
			cookieH = cookie.Handler(next)
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.HasPrefix(r.URL.Path, "/collect/"):
				collectH.ServeHTTP(w, r)
			case cookiePath(r.URL.Path):
				cookieH.ServeHTTP(w, r)
			default:
				openH.ServeHTTP(w, r)
			}
		})
	}
}

// crossOriginGuard rejects cross-site unsafe requests (Sec-Fetch-Site, falling back
// to Origin vs Host) on the cookie-authenticated paths with the stdlib's
// http.CrossOriginProtection. This is why go-pkgz's own XSRF double-submit stays
// disabled: it would need a token round-trip in the generated client for the same
// protection. Bearer and secret-key surfaces (collect, API, MCP, OAuth, hooks,
// tracking) are cross-origin by design and pass through untouched.
//
// The app origin (APP_URL, which may carry a path or trailing slash) and the
// credentialed-CORS allowlist are trusted: an origin allowed to send cookies
// cross-origin must also be allowed to write.
func crossOriginGuard(appURL string, allowed []string) (func(http.Handler) http.Handler, error) {
	cop := http.NewCrossOriginProtection()
	for _, raw := range append([]string{appURL}, allowed...) {
		u, err := url.Parse(raw)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("trusted origin %q: want scheme://host", raw)
		}
		if err := cop.AddTrustedOrigin(u.Scheme + "://" + u.Host); err != nil {
			return nil, fmt.Errorf("trusted origin %q: %w", raw, err)
		}
	}
	return func(next http.Handler) http.Handler {
		guarded := cop.Handler(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cookiePath(r.URL.Path) {
				guarded.ServeHTTP(w, r)
				return
			}
			next.ServeHTTP(w, r)
		})
	}, nil
}
