package inertia

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// CSRF is a middleware for protecting cross-site request forgery with Inertia.js
// See: https://inertiajs.com/csrf-protection
var CSRF = func() echo.MiddlewareFunc {
	return middleware.CSRFWithConfig(middleware.CSRFConfig{
		TokenLookup: "header:X-XSRF-TOKEN",
		CookieName:  "XSRF-TOKEN",
		CookiePath:  "/",
	})
}
