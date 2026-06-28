package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
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
	"github.com/mokevnin/1mail/internal/messaging/registry"
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/mokevnin/1mail/internal/secrets"
	"github.com/mokevnin/1mail/internal/tracking"
	"github.com/ogen-go/ogen/ogenerrors"
)

// New builds the top-level net/http handler wiring the three ogen-generated
// API servers (site, external, collect) plus go-pkgz/auth endpoints.
func New(cfg *config.Config, client *ent.Client, ps *pubsub.PubSub, enqueuer apisite.BroadcastEnqueuer) (http.Handler, error) {
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
		apisite.NewHandlers(client, ps, cipher, providerCatalog, enqueuer),
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
		apiexternal.NewHandlers(client, cfg.BootstrapToken),
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
		apicollect.NewHandlers(client),
		apiauth.NewCollectSecurityHandler(client),
		collectapi.WithPathPrefix("/collect"),
		collectapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/collect/", colSrv)

	// Public tracking snippet (no auth) — embedded IIFE bundle.
	mux.Handle("/t.js", trackerHandler())

	// Public email engagement endpoints (open pixel, click redirect, unsubscribe).
	mux.Handle("/e/", trackingHandler(client, tracking.New(cfg.JWTSecret, cfg.AppURL)))

	// Catch-all: the embedded SPA (release builds with -tags embed_spa). Most
	// specific pattern wins, so this never shadows the API prefixes above.
	mux.Handle("/", spaHandler())

	return chain(mux, recoverer, requestID, timeout(30*time.Second), cors(cfg.CORSOrigins)), nil
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
		next.ServeHTTP(w, r)
	})
}

func randomID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "req"
	}
	return hex.EncodeToString(b)
}

func cors(origins []string) func(http.Handler) http.Handler {
	allowed := make(map[string]struct{}, len(origins))
	for _, o := range origins {
		allowed[o] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin != "" {
				if strings.HasPrefix(r.URL.Path, "/collect/") {
					// Public ingestion from arbitrary customer sites: the collect
					// key is public, so allow any origin. No credentials (cookies
					// are first-party on the customer's own domain), which lets us
					// echo the origin instead of being stuck with a static list.
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Vary", "Origin")
				} else if _, ok := allowed[origin]; ok || len(origins) == 0 {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Vary", "Origin")
				}
			}
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", strings.Join([]string{"Authorization", "Content-Type", "x-collect-key", "x-bootstrap-token"}, ","))
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
