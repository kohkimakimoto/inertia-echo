package inertia

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CSRF is a middleware for protecting cross-site request forgery with Inertia.js
// See: https://inertiajs.com/csrf-protection

type CSRFConfig middleware.CSRFConfig

var DefaultCSRFConfig = CSRFConfig{
	Skipper:        middleware.DefaultSkipper,
	TokenLength:    32,
	TokenLookup:    "header:X-XSRF-TOKEN",
	ContextKey:     "csrf",
	CookieName:     "XSRF-TOKEN",
	CookieMaxAge:   86400,
	CookieSameSite: http.SameSiteDefaultMode,
	CookiePath:     "/",
}

func CSRF() echo.MiddlewareFunc {
	return CSRFWithConfig(DefaultCSRFConfig)
}

func CSRFWithConfig(config CSRFConfig) echo.MiddlewareFunc {
	if config.Skipper == nil {
		config.Skipper = DefaultCSRFConfig.Skipper
	}
	if config.TokenLength == 0 {
		config.TokenLength = DefaultCSRFConfig.TokenLength
	}
	if config.TokenLookup == "" {
		config.TokenLookup = DefaultCSRFConfig.TokenLookup
	}
	if config.ContextKey == "" {
		config.ContextKey = DefaultCSRFConfig.ContextKey
	}
	if config.CookieName == "" {
		config.CookieName = DefaultCSRFConfig.CookieName
	}
	if config.CookieMaxAge == 0 {
		config.CookieMaxAge = DefaultCSRFConfig.CookieMaxAge
	}
	if config.CookieSameSite == http.SameSiteNoneMode {
		config.CookieSecure = true
	}
	if config.CookiePath == "" {
		config.CookiePath = DefaultCSRFConfig.CookiePath
	}

	return middleware.CSRFWithConfig(middleware.CSRFConfig(config))
}
