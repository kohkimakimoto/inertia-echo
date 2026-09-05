package inertia

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v5"
)

// WrapHTTPErrorHandler applies the same Inertia redirect and one-shot state
// finalization used by MiddlewareWithConfig to responses created by Echo's
// central error handler.
func WrapHTTPErrorHandler(handler echo.HTTPErrorHandler) echo.HTTPErrorHandler {
	return func(c *echo.Context, handlerErr error) {
		if handler == nil {
			return
		}
		if response, _ := echo.UnwrapResponse(c.Response()); response != nil && response.Committed {
			return
		}

		original := c.Response()
		writer := NewResponseWriterWrapper(original)
		c.SetResponse(writer)
		defer c.SetResponse(original)
		addVaryHeader(writer.Header(), HeaderXInertia)

		handler(c, handlerErr)
		if i, err := Get(c); err == nil {
			if writer.isTransitionResponse() && i.reflash != nil {
				if err := i.reflash(c); err != nil {
					c.Logger().Error("inertia error handler failed to reflash data", "error", err)
					return
				}
			}
			i.sendClearHistoryCookieIfNeeded()
			i.sendPreserveFragmentCookieIfNeeded()
		}
		transformResponse(c.Request(), writer)
		if err := writer.flushBufferedResponse(); err != nil {
			c.Logger().Error("inertia error handler failed to write response", "error", err)
		}
	}
}

type ErrorPagePropsFunc func(c *echo.Context, status int, err error) map[string]any

// ErrorHandlerConfig configures an opt-in Inertia error page handler.
type ErrorHandlerConfig struct {
	Middleware    MiddlewareConfig
	Component     string
	Statuses      []int
	Props         ErrorPagePropsFunc
	ResolveShared bool
	Fallback      echo.HTTPErrorHandler
}

// HTTPErrorHandlerWithConfig renders selected HTTP errors as an Inertia page.
// When no Inertia context exists it initializes a minimal request context from
// the supplied middleware config. Shared props are reloaded only when
// ResolveShared is true.
func HTTPErrorHandlerWithConfig(config ErrorHandlerConfig) echo.HTTPErrorHandler {
	if config.Component == "" {
		config.Component = "ErrorPage"
	}
	if len(config.Statuses) == 0 {
		config.Statuses = []int{http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError, http.StatusServiceUnavailable}
	}
	if config.Fallback == nil {
		config.Fallback = echo.DefaultHTTPErrorHandler(false)
	}

	base := func(c *echo.Context, handlerErr error) {
		status := http.StatusInternalServerError
		var coder echo.HTTPStatusCoder
		if errors.As(handlerErr, &coder) && coder.StatusCode() != 0 {
			status = coder.StatusCode()
		}
		if !inIntArray(status, config.Statuses) {
			config.Fallback(c, handlerErr)
			return
		}

		i, err := inertiaForErrorPage(c, config.Middleware, config.ResolveShared)
		if err != nil {
			config.Fallback(c, err)
			return
		}
		props := map[string]any{"status": status}
		if config.Props != nil {
			if configured := config.Props(c, status, handlerErr); configured != nil {
				props = configured
			}
		}
		if err := i.RenderWithStatus(status, config.Component, props); err != nil {
			config.Fallback(c, err)
		}
	}

	return WrapHTTPErrorHandler(base)
}

func inertiaForErrorPage(c *echo.Context, config MiddlewareConfig, resolveShared bool) (*Inertia, error) {
	if i, err := Get(c); err == nil {
		return i, nil
	}
	applyMiddlewareDefaults(&config)
	shared := map[string]any{}
	if resolveShared && config.Share != nil {
		resolved, err := config.Share(c)
		if err != nil {
			return nil, err
		}
		if resolved != nil {
			shared = resolved
		}
	}
	i := newInertia(c, config, shared)
	c.Set(key, i)
	return i, nil
}

func inIntArray(value int, values []int) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
