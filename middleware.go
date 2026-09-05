package inertia

import (
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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
	// PreserveFragmentCookieKey stores the one-shot preserve-fragment flag.
	PreserveFragmentCookieKey string
	// FlashData loads application-owned flash data when a Page is rendered.
	FlashData FlashDataFunc
	// Reflash keeps application-owned flash data across redirects and 409 visits.
	Reflash ReflashFunc
	// RescueReporter receives errors from deferred props with Rescue enabled.
	RescueReporter RescueReporter
	// Now supplies request-time timestamps for once-prop expiration metadata.
	Now func() time.Time
	// IsSsrDisabled is a flag that determines whether server-side rendering is disabled.
	// If this is true, server-side rendering is disabled even if the renderer supports and is configured for it.
	IsSsrDisabled bool
}

type SharedDataFunc func(c *echo.Context) (map[string]any, error)
type FlashDataFunc func(c *echo.Context) (map[string]any, error)
type ReflashFunc func(c *echo.Context) error

var DefaultMiddlewareConfig = MiddlewareConfig{
	Skipper:                   middleware.DefaultSkipper,
	RootView:                  "app.html",
	VersionFunc:               defaultVersionFunc(),
	Share:                     nil,
	Renderer:                  nil,
	ClearHistoryCookieKey:     "inertia.clear_history",
	PreserveFragmentCookieKey: "inertia.preserve_fragment",
	IsSsrDisabled:             false,
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
	applyMiddlewareDefaults(&config)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) (err error) {
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
			}
			if sharedProps == nil {
				sharedProps = map[string]any{}
			}

			i := newInertia(c, config, sharedProps)
			c.Set(key, i)

			req := c.Request()

			// In the event that the assets change, initiate a
			// client-side location visit to force an update.
			// see https://inertiajs.com/the-protocol#asset-versioning
			version := i.Version()
			if req.Header.Get(HeaderXInertia) != "" && checkVersion(req, version) {
				addVaryHeader(c.Response().Header(), HeaderXInertia)
				if i.reflash != nil {
					if err := i.reflash(c); err != nil {
						return err
					}
				}
				i.sendClearHistoryCookieIfNeeded()
				i.sendPreserveFragmentCookieIfNeeded()
				c.Response().Header().Set(HeaderXInertiaVersion, version)
				return i.Location(req.URL.String())
			}

			// Wrap the http response writer.
			// The response status code might change after the handler executes.
			originalWriter := c.Response()
			w := NewResponseWriterWrapper(originalWriter)
			c.SetResponse(w)
			defer c.SetResponse(originalWriter)

			if err = next(c); err != nil {
				return
			}
			addVaryHeader(w.Header(), HeaderXInertia)

			if w.isTransitionResponse() && i.reflash != nil {
				if err = i.reflash(c); err != nil {
					return
				}
			}
			i.sendClearHistoryCookieIfNeeded()
			i.sendPreserveFragmentCookieIfNeeded()
			transformResponse(req, w)
			return w.flushBufferedResponse()
		}
	}
}

func applyMiddlewareDefaults(config *MiddlewareConfig) {
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
	if config.PreserveFragmentCookieKey == "" {
		config.PreserveFragmentCookieKey = DefaultMiddlewareConfig.PreserveFragmentCookieKey
	}
}

func newInertia(c *echo.Context, config MiddlewareConfig, sharedProps map[string]any) *Inertia {
	if sharedProps == nil {
		sharedProps = map[string]any{}
	}
	i := &Inertia{
		echoContext:               c,
		rootView:                  config.RootView,
		sharedProps:               sharedProps,
		version:                   config.VersionFunc,
		renderer:                  config.Renderer,
		clearHistoryCookieKey:     config.ClearHistoryCookieKey,
		preserveFragmentCookieKey: config.PreserveFragmentCookieKey,
		flash:                     map[string]any{},
		flashData:                 config.FlashData,
		reflash:                   config.Reflash,
		rescueReporter:            config.RescueReporter,
		now:                       config.Now,
		isSsrDisabled:             config.IsSsrDisabled,
	}
	req := c.Request()
	i.partialComponent = req.Header.Get(HeaderXInertiaPartialComponent)
	i.onlyProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialData), ",")
	i.exceptProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialExcept), ",")
	i.resetProps = splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaReset), ",")
	return i
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
func changeRedirectCode(req *http.Request, w *ResponseWriterWrapper) {
	if req.Header.Get(HeaderXInertia) != "" &&
		inArray(req.Method, []string{http.MethodPut, http.MethodPatch, http.MethodDelete}) {
		w.replaceBufferedStatusCode(http.StatusFound, http.StatusSeeOther)
	}
}

func transformResponse(req *http.Request, w *ResponseWriterWrapper) {
	if req.Header.Get(HeaderXInertia) != "" && req.Header.Get("Purpose") != "prefetch" && w.hasFragmentRedirect() {
		location := w.Header().Get(echo.HeaderLocation)
		w.replaceBufferedResponse(http.StatusConflict, nil)
		w.Header().Del(echo.HeaderLocation)
		w.Header().Del(echo.HeaderContentLength)
		w.Header().Del(echo.HeaderContentType)
		w.Header().Set(HeaderXInertiaRedirect, location)
		return
	}
	changeRedirectCode(req, w)
}

func Get(c *echo.Context) (*Inertia, error) {
	in, ok := c.Get(key).(*Inertia)
	if !ok {
		return nil, ErrNoInertiaContext
	}
	return in, nil
}

func MustGet(c *echo.Context) *Inertia {
	in, err := Get(c)
	if err != nil {
		panic(err)
	}
	return in
}

func Has(c *echo.Context) bool {
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

		return func(c *echo.Context) (err error) {
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
