package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	"github.com/mokevnin/1mail/internal/authtoken"
	"github.com/mokevnin/1mail/internal/events"
	"github.com/mokevnin/1mail/internal/logging"
	"github.com/mokevnin/1mail/internal/messaging/registry"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/telemetry"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/ogen-go/ogen/ogenerrors"
	"github.com/rs/cors"
)

// New builds the top-level net/http handler wiring the three ogen-generated
// API servers (site, external, collect) plus go-pkgz/auth endpoints.
func New(cfg *config.Config, client *ent.Client, db *sql.DB, bus *events.Bus, enqueuer apisite.BroadcastEnqueuer, welcome apisite.WelcomeEnqueuer, sysmail apisite.SystemMailEnqueuer, resolver apiexternal.SenderResolver) (http.Handler, error) {
	// Credential encryption is mandatory: fail fast at boot if the key is
	// missing or malformed rather than at first provider write.
	cipher, err := secrets.NewCipher(cfg.EncryptionKey)
	if err != nil {
		return nil, fmt.Errorf("encryption key: %w", err)
	}
	providerCatalog := registry.Default()

	mux := http.NewServeMux()

	// Auth service (go-pkgz/auth) — JWT issuance + direct (email/password) provider.
	authSvc := goauth.NewService(goauth.Opts{
		SecretReader:   token.SecretFunc(func(string) (string, error) { return cfg.JWTSecret, nil }),
		TokenDuration:  time.Hour,
		CookieDuration: 24 * time.Hour,
		DisableXSRF:    true,
		SecureCookies:  false,
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
		apisite.NewHandlers(client, bus, cipher, providerCatalog, enqueuer, welcome, sysmail, authtoken.New(cfg.JWTSecret), cfg.AppURL),
		apiauth.NewSiteSecurityHandler(cfg.JWTSecret, client),
		siteapi.WithPathPrefix("/site"),
		siteapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/site/", siteSrv)

	// External API — /api (Bearer token auth via ogen SecurityHandler).
	extSrv, err := externalapi.NewServer(
		apiexternal.NewHandlers(client, cfg.BootstrapToken, bus, resolver),
		apiauth.NewExternalSecurityHandler(client),
		externalapi.WithPathPrefix("/api"),
		externalapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/api/", extSrv)

	// Collect API — /collect (x-collect-key via generated SecurityHandler).
	colSrv, err := collectapi.NewServer(
		apicollect.NewHandlers(client, bus),
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
	mux.Handle("/e/", trackingHandler(client, bus, tracking.New(cfg.JWTSecret, cfg.AppURL)))

	// Inbound provider webhooks (SES bounce/complaint via SNS), routed by the
	// workspace's secret ingest key: POST /hooks/{key}/{provider}.
	mux.Handle("/hooks/", hooksHandler(client, bus))

	// Catch-all: the embedded SPA (release builds with -tags embed_spa). Most
	// specific pattern wins, so this never shadows the API prefixes above.
	mux.Handle("/", spaHandler())

	// requestID is outermost so the correlation id is in context before recoverer
	// runs — the panic log then carries request_id. (requestID is trivial and
	// cannot itself panic, so nothing downstream of recovery is lost.)
	return chain(mux, requestID, recoverer, timeout(30*time.Second), corsMiddleware(cfg.CORSOrigins)), nil
}

// problemErrorHandler renders ogen errors as RFC 7807 application/problem+json.
func problemErrorHandler(_ context.Context, w http.ResponseWriter, _ *http.Request, err error) {
	code := http.StatusInternalServerError
	var oe ogenerrors.Error
	if errors.As(err, &oe) {
		code = oe.Code()
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
		defer func() {
			if rec := recover(); rec != nil {
				logging.FromContext(r.Context()).Error("panic recovered",
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

func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = randomID()
		}
		w.Header().Set("X-Request-Id", id)
		// Carry the id in context so request-scoped logs (via logging.FromContext)
		// correlate back to this request.
		next.ServeHTTP(w, r.WithContext(logging.WithRequestID(r.Context(), id)))
	})
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req"
	}
	return hex.EncodeToString(b)
}

// corsMiddleware applies two rs/cors policies by path: the public collect API
// echoes any origin without credentials (the collect key is public, cookies are
// first-party on the customer's own domain), while the rest of the app uses the
// configured allowlist with credentials (an empty list reflects any origin).
func corsMiddleware(origins []string) func(http.Handler) http.Handler {
	app := cors.New(cors.Options{
		AllowedMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions,
		},
		// "*" reflects whatever request headers the client asks for (works with
		// credentials, and avoids brittle exact-match of comma-joined header lists).
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
		AllowedOrigins:   origins,
		// Empty allowlist ⇒ reflect any origin (dev convenience); AllowedOrigins
		// is ignored when AllowOriginFunc is set, so only install it when empty.
		AllowOriginFunc: reflectAllWhenEmpty(origins),
	})
	collect := cors.New(cors.Options{
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: false,
		AllowOriginFunc:  func(string) bool { return true },
	})

	return func(next http.Handler) http.Handler {
		appH := app.Handler(next)
		collectH := collect.Handler(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/collect/") {
				collectH.ServeHTTP(w, r)
				return
			}
			appH.ServeHTTP(w, r)
		})
	}
}

// reflectAllWhenEmpty returns an AllowOriginFunc that reflects any origin when no
// allowlist is configured, or nil so rs/cors uses AllowedOrigins otherwise.
func reflectAllWhenEmpty(origins []string) func(string) bool {
	if len(origins) > 0 {
		return nil
	}
	return func(string) bool { return true }
}
