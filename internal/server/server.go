package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/mokevnin/1mail/internal/pubsub"
	"github.com/ogen-go/ogen/ogenerrors"
)

// New builds the top-level net/http handler wiring the three ogen-generated
// API servers (site, external, collect) plus go-pkgz/auth endpoints.
func New(cfg *config.Config, client *ent.Client, ps *pubsub.PubSub) (http.Handler, error) {
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

	// Site API — /site (JWT cookie via generated SecurityHandler; register and
	// direct-login are public per the spec).
	siteSrv, err := siteapi.NewServer(
		apisite.NewHandlers(client, ps),
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
		apiauth.NewCollectSecurityHandler(cfg.CollectSiteKey),
		collectapi.WithPathPrefix("/collect"),
		collectapi.WithErrorHandler(problemErrorHandler),
	)
	if err != nil {
		return nil, err
	}
	mux.Handle("/collect/", colSrv)

	// Public tracking snippet (no auth) — embedded IIFE bundle.
	mux.Handle("/t.js", trackerHandler())

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
