package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-pkgz/auth/v2"
	"github.com/go-pkgz/auth/v2/avatar"
	"github.com/go-pkgz/auth/v2/token"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mokevnin/1mail/config"
	"github.com/mokevnin/1mail/ent"
	collectapi "github.com/mokevnin/1mail/gen/collect"
	externalapi "github.com/mokevnin/1mail/gen/external"
	siteapi "github.com/mokevnin/1mail/gen/site"
	apicollect "github.com/mokevnin/1mail/internal/api/collect"
	apiexternal "github.com/mokevnin/1mail/internal/api/external"
	apisite "github.com/mokevnin/1mail/internal/api/site"
	"github.com/mokevnin/1mail/internal/pubsub"
)

func New(cfg *config.Config, client *ent.Client, ps *pubsub.PubSub) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.Gzip())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     cfg.CORSOrigins,
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders:     []string{echo.HeaderAuthorization, echo.HeaderContentType},
		AllowCredentials: true,
	}))
	e.Use(middleware.ContextTimeout(30 * time.Second))

	e.HTTPErrorHandler = problemErrorHandler

	// Auth service (go-pkgz/auth)
	authSvc := auth.NewService(auth.Opts{
		SecretReader:   token.SecretFunc(func(string) (string, error) { return cfg.JWTSecret, nil }),
		TokenDuration:  time.Hour,
		CookieDuration: 24 * time.Hour,
		DisableXSRF:    true,
		SecureCookies:  false,
		Issuer:         "1mail",
		URL:            cfg.AppURL,
		AvatarStore:    avatar.NewLocalFS("/tmp/1mail-avatars"),
	})
	authSvc.AddDirectProvider("direct", apisite.NewCredChecker(client))

	authHandler, avatarHandler := authSvc.Handlers()
	e.Any("/auth/*", echo.WrapHandler(authHandler))
	e.GET("/avatar/*", echo.WrapHandler(avatarHandler))

	m := authSvc.Middleware()

	// Site API — /site (JWT auth, register endpoint is public)
	siteGroup := e.Group("/site")
	siteGroup.Use(echo.WrapMiddleware(skipAuth(m.Auth, "/site/auth/register")))
	siteHandlers := siteapi.NewStrictHandler(apisite.NewHandlers(client, ps), nil)
	siteapi.RegisterHandlers(siteGroup, siteHandlers)

	// External API — /api (Bearer token auth)
	extGroup := e.Group("/api")
	extGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:  "header:Authorization",
		AuthScheme: "Bearer",
		Validator:  apiexternal.MakeTokenValidator(client),
		Skipper: func(c echo.Context) bool {
			return c.Path() == "/api/auth/tokens/bootstrap"
		},
	}))
	extGroup.Use(middleware.BodyLimit("1M"))
	externalHandlers := externalapi.NewStrictHandler(apiexternal.NewHandlers(client, cfg.BootstrapToken), nil)
	externalapi.RegisterHandlers(extGroup, externalHandlers)

	// Collect API — x-collect-key
	collectGroup := e.Group("/collect")
	collectGroup.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:x-collect-key",
		Validator: apiexternal.MakeCollectKeyValidator(cfg.CollectSiteKey),
	}))
	collectHandlers := collectapi.NewStrictHandler(apicollect.NewHandlers(client), nil)
	collectapi.RegisterHandlers(collectGroup, collectHandlers)

	return e
}

// skipAuth wraps an auth middleware but bypasses it for specified exact paths.
func skipAuth(authMW func(http.Handler) http.Handler, paths ...string) func(http.Handler) http.Handler {
	skip := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		skip[p] = struct{}{}
	}
	return func(next http.Handler) http.Handler {
		protected := authMW(next)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := skip[r.URL.Path]; ok {
				next.ServeHTTP(w, r)
				return
			}
			protected.ServeHTTP(w, r)
		})
	}
}

func problemErrorHandler(err error, c echo.Context) {
	code := http.StatusInternalServerError
	var msg string

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		if m, ok := he.Message.(string); ok {
			msg = m
		} else {
			b, _ := json.Marshal(he.Message)
			msg = string(b)
		}
	} else {
		msg = err.Error()
	}

	if !c.Response().Committed {
		c.Response().Header().Set(echo.HeaderContentType, "application/problem+json")
		status := int32(code)
		prob := struct {
			Status int32   `json:"status"`
			Title  *string `json:"title,omitempty"`
			Detail *string `json:"detail,omitempty"`
		}{
			Status: status,
			Detail: &msg,
		}
		title := http.StatusText(code)
		prob.Title = &title
		_ = c.JSON(code, prob)
	}
}
