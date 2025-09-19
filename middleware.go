package inertia

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

const (
	key = "__inertia__"
)

type MiddlewareConfig struct {
	Skipper middleware.Skipper
	// The root template that's loaded on the first page visit.
	// see https://inertiajs.com/server-side-setup#root-template
	RootView string
	// Determines the current asset version.
	// see https://inertiajs.com/asset-versioning
	VersionFunc func() string
	// Defines the props that are shared by default.
	// see https://inertiajs.com/shared-data
	Share SharedDataFunc
	// Renderer is a renderer that is used for rendering the root view.
	Renderer Renderer
	// ClearHistoryCookieKey is a key for the cookie that is used to clear the history state.
	ClearHistoryCookieKey string
	// IsSsrDisabled is a flag that determines whether server-side rendering is disabled.
	// If this is true, server-side rendering is disabled even if the renderer supports and is configured for it.
	IsSsrDisabled bool
}

type SharedDataFunc func(c echo.Context) (map[string]any, error)

var DefaultMiddlewareConfig = MiddlewareConfig{
	Skipper:               middleware.DefaultSkipper,
	RootView:              "app.html",
	VersionFunc:           defaultVersionFunc(),
	Share:                 nil,
	Renderer:              nil,
	ClearHistoryCookieKey: "inertia.clear_history",
	IsSsrDisabled:         false,
}

func defaultVersionFunc() VersionFunc {
	var v string

	if v = os.Getenv("INERTIA_VERSION"); v == "" {
		// `GAE_VERSION` is for Google App Engine.
		// see https://cloud.google.com/appengine/docs/standard/go/runtime#environment_variables
		if v = os.Getenv("GAE_VERSION"); v == "" {
			// The fallback version value that imitates the default GAE version format.
			// It assumes to be used for development.
			v = time.Now().Format("20060102t150405")
		}
	}

	return func() string {
		return v
	}
}

// MiddlewareWithConfig returns an echo middleware that adds the Inertia instance to the context.
func MiddlewareWithConfig(config MiddlewareConfig) echo.MiddlewareFunc {
	// Defaults
	if config.Skipper == nil {
		config.Skipper = DefaultMiddlewareConfig.Skipper
	}
	if config.RootView == "" {
		config.RootView = DefaultMiddlewareConfig.RootView
	}
	if config.VersionFunc == nil {
		config.VersionFunc = DefaultMiddlewareConfig.VersionFunc
	}
	if config.ClearHistoryCookieKey == "" {
		config.ClearHistoryCookieKey = DefaultMiddlewareConfig.ClearHistoryCookieKey
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			var sharedProps map[string]any
			if config.Share != nil {
				ret, err := config.Share(c)
				if err != nil {
					return err
				}
				sharedProps = ret
			} else {
				sharedProps = map[string]any{}
			}

			// Create an Inertia instance.
			i := &Inertia{
				echoContext:           c,
				rootView:              config.RootView,
				sharedProps:           sharedProps,
				version:               config.VersionFunc,
				renderer:              config.Renderer,
				clearHistoryCookieKey: config.ClearHistoryCookieKey,
				isSsrDisabled:         config.IsSsrDisabled,
			}
			c.Set(key, i)

			req := c.Request()
			res := c.Response()

			i.partialComponent = req.Header.Get(HeaderXInertiaPartialComponent)
			i.onlyProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialData), ",")
			i.exceptProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialExcept), ",")
			i.resetProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaReset), ",")

			if req.Header.Get(HeaderXInertia) == "" {
				// Not inertial request
				if err = next(c); err != nil {
					return
				}
				i.sendClearHistoryCookieIfNeeded()

				return
			}

			// In the event that the assets change, initiate a
			// client-side location visit to force an update.
			// see https://inertiajs.com/the-protocol#asset-versioning
			if checkVersion(req, i.Version()) {
				err = i.Location(req.URL.Path)
				return
			}

			// Wrap the http response writer.
			// The response status code might change after the handler executes.
			w := NewResponseWriterWrapper(res.Writer)
			res.Writer = w
			defer func(w *ResponseWriterWrapper) {
				// send buffered header and restore the original response writer
				w.FlushHeader()
				res.Writer = w.ResponseWriter
			}(w)

			if err = next(c); err != nil {
				return
			}
			i.sendClearHistoryCookieIfNeeded()

			changeRedirectCode(req, res)
			return
		}
	}
}

// checkVersion checks the assets version change.
func checkVersion(req *http.Request, version string) bool {
	if req.Header.Get(HeaderXInertia) != "" &&
		req.Method == "GET" &&
		req.Header.Get(HeaderXInertiaVersion) != version {
		return true
	}
	return false
}

// changeRedirectCode changes the status code during redirects, ensuring they are made as
// GET requests, preventing "MethodNotAllowedHttpException" errors.
// see https://inertiajs.com/redirects
func changeRedirectCode(req *http.Request, res *echo.Response) {
	if req.Header.Get(HeaderXInertia) != "" &&
		res.Status == 302 &&
		inArray(req.Method, []string{"PUT", "PATCH", "DELETE"}) {
		res.Status = 303
		res.Writer.WriteHeader(303)
	}
}

func Get(c echo.Context) (*Inertia, error) {
	in, ok := c.Get(key).(*Inertia)
	if !ok {
		return nil, ErrNoInertiaContext
	}
	return in, nil
}

func MustGet(c echo.Context) *Inertia {
	in, err := Get(c)
	if err != nil {
		panic(err)
	}
	return in
}

func Has(c echo.Context) bool {
	_, ok := c.Get(key).(*Inertia)
	return ok
}

type EncryptHistoryMiddlewareConfig struct {
	Skipper middleware.Skipper
}

func EncryptHistoryMiddleware() echo.MiddlewareFunc {
	return EncryptHistoryMiddlewareWithConfig(EncryptHistoryMiddlewareConfig{})
}

func EncryptHistoryMiddlewareWithConfig(config EncryptHistoryMiddlewareConfig) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		if config.Skipper == nil {
			config.Skipper = middleware.DefaultSkipper
		}

		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			i, err := Get(c)
			if err != nil {
				return err
			}

			i.EncryptHistory(true)

			return next(c)
		}
	}
}
