package server

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mokevnin/1mail/backend/config"
	"github.com/mokevnin/1mail/backend/ent"
	collectapi "github.com/mokevnin/1mail/backend/gen/collect"
	externalapi "github.com/mokevnin/1mail/backend/gen/external"
	siteapi "github.com/mokevnin/1mail/backend/gen/site"
	apicollect "github.com/mokevnin/1mail/backend/internal/api/collect"
	apiexternal "github.com/mokevnin/1mail/backend/internal/api/external"
	apisite "github.com/mokevnin/1mail/backend/internal/api/site"
)

func New(cfg *config.Config, client *ent.Client) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.RequestID())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.Secure())
	e.Use(middleware.Gzip())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.CORSOrigins,
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderAuthorization, echo.HeaderContentType},
	}))
	e.Use(middleware.ContextTimeout(30 * time.Second))

	e.HTTPErrorHandler = problemErrorHandler

	// Site API — /site (no auth)
	siteGroup := e.Group("/site")
	siteHandlers := siteapi.NewStrictHandler(apisite.NewHandlers(client), nil)
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
