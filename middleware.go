package inertia

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const (
	key = "__inertia__"
)

type SharedDataFunc func(c *echo.Context) (map[string]any, error)
type FlashDataFunc func(c *echo.Context) (map[string]any, error)
type ReflashFunc func(c *echo.Context) error

type MiddlewareConfig struct {
	Skipper middleware.Skipper
	// The root template that's loaded on the first page visit.
	// Corresponding specification: https://inertiajs.com/docs/v3/installation/server-side-setup
	// Unlike inertia-laravel's rootView(Request), this stays a static string: in practice
	// a fixed template name is enough. Override per request with SetRootView if needed.
	RootView string
	// Determines the current asset version.
	// Corresponding specification: https://inertiajs.com/docs/v3/advanced/asset-versioning
	// Signature mirrors inertia-laravel's version(Request); the default ignores c.
	VersionFunc VersionFunc
	// URLResolver customizes the Page object URL. nil uses req.URL.String().
	// Corresponding specification: https://inertiajs.com/docs/v3/core-concepts/the-protocol#the-page-object
	// Mirrors inertia-laravel's resolveUrlUsing / Middleware::urlResolver.
	URLResolver URLResolverFunc
	// Defines the props that are shared by default.
	// Corresponding specification: https://inertiajs.com/docs/v3/data-props/shared-data
	Share SharedDataFunc
	// Defines once props shared on every request. Map values should be
	// func() (any, error) or *OnceProp (same contract as Laravel's shareOnce).
	// Corresponding specification: https://inertiajs.com/docs/v3/data-props/shared-data#sharing-once-props
	ShareOnce SharedDataFunc
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
	// WithoutSsr excludes matching paths from SSR for each request.
	// Mirrors inertia-laravel Middleware::$withoutSsr.
	// Patterns support * wildcards (Laravel Request::is / ExcludesPaths style).
	// Stored on the request-scoped Inertia (not mutated on a shared SsrGateway) for Go concurrency safety.
	WithoutSsr []string
}

var DefaultMiddlewareConfig = MiddlewareConfig{
	Skipper:                   middleware.DefaultSkipper,
	RootView:                  "app.html",
	VersionFunc:               defaultVersionFunc(),
	Share:                     nil,
	Renderer:                  nil,
	ClearHistoryCookieKey:     "inertia.clear_history",
	PreserveFragmentCookieKey: "inertia.preserve_fragment",
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

	return func(*echo.Context) string {
		return v
	}
}

// MiddlewareWithConfig returns an echo middleware that adds the Inertia instance to the context.
func MiddlewareWithConfig(config MiddlewareConfig) echo.MiddlewareFunc {
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

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c *echo.Context) error {
			if config.Skipper(c) {
				return next(c)
			}

			// Create the request-scoped Inertia instance,
			req := c.Request()
			i := &Inertia{
				echoContext:               c,
				rootView:                  config.RootView,
				sharedProps:               map[string]any{},
				version:                   config.VersionFunc,
				urlResolver:               config.URLResolver,
				renderer:                  config.Renderer,
				clearHistoryCookieKey:     config.ClearHistoryCookieKey,
				preserveFragmentCookieKey: config.PreserveFragmentCookieKey,
				flash:                     map[string]any{},
				flashData:                 config.FlashData,
				reflash:                   config.Reflash,
				rescueReporter:            config.RescueReporter,
				now:                       config.Now,
				partialComponent:          req.Header.Get(HeaderXInertiaPartialComponent),
				onlyProps:                 splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialData), ","),
				exceptProps:               splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaPartialExcept), ","),
				resetProps:                splitAndRemoveEmpty(req.Header.Get(HeaderXInertiaReset), ","),
			}
			c.Set(key, i)

			if config.Share != nil {
				shared, err := config.Share(c)
				if err != nil {
					return err
				}
				if shared != nil {
					i.Share(shared)
				}
			}
			if config.ShareOnce != nil {
				onceShared, err := config.ShareOnce(c)
				if err != nil {
					return err
				}
				for propKey, value := range onceShared {
					switch v := value.(type) {
					case *OnceProp:
						i.Share(map[string]any{propKey: v})
					case func() (any, error):
						i.ShareOnce(propKey, v)
					default:
						return fmt.Errorf("inertia-echo: ShareOnce %q: want func() (any, error) or *OnceProp, got %T", propKey, value)
					}
				}
			}

			// Mirrors inertia-laravel Middleware::handle: withoutSsr is applied after
			// share/shareOnce (and rootView/urlResolver, set on Inertia at construction)
			// and immediately before $next. Stored on request-scoped Inertia instead of
			// mutating a shared SsrGateway (Go concurrency).
			if len(config.WithoutSsr) > 0 {
				i.withoutSsr = append([]string(nil), config.WithoutSsr...)
			}

			// Buffer the full response for Inertia requests so post-next rewrites
			// (version change, empty response, 303, fragment 409) can replace it,
			// matching inertia-laravel Middleware::handle after $next($request).
			isInertia := req.Header.Get(HeaderXInertia) != ""
			originalWriter := c.Response()
			w := NewResponseWriterWrapperWithOptions(originalWriter, isInertia)
			c.SetResponse(w)
			defer c.SetResponse(originalWriter)

			if err := next(c); err != nil {
				// Propagate to the calling middleware or Echo's HTTPErrorHandler.
				// This package intentionally does not provide an inertia-laravel's style
				// error-handling framework. Echo and Go favor explicit application logic.
				// Typically, you should handle errors in your application's echo.HTTPErrorHandler,
				// and if you want to render an Inertia error page, you can do so there.
				return err
			}

			// Post-handler pipeline mirrors inertia-laravel Middleware::handle after $next.
			addVaryHeader(w.Header(), HeaderXInertia)

			if w.isRedirectResponse() && i.reflash != nil {
				if err := i.reflash(c); err != nil {
					return err
				}
			}

			if !isInertia {
				i.sendClearHistoryCookieIfNeeded()
				i.sendPreserveFragmentCookieIfNeeded()
				return w.flushBufferedResponse()
			}

			// Corresponding specification: https://inertiajs.com/docs/v3/core-concepts/the-protocol#asset-versioning
			version := i.Version()
			if checkVersion(req, version) {
				if err := onVersionChange(c, i, w, version); err != nil {
					return err
				}
				return w.flushBufferedResponse()
			}

			// Mirrors inertia-laravel onEmptyResponse: empty 200 → redirect back.
			if w.isEmptyOKResponse() {
				if err := onEmptyResponse(c, w); err != nil {
					return err
				}
			}

			transformResponse(req, w)
			i.sendClearHistoryCookieIfNeeded()
			i.sendPreserveFragmentCookieIfNeeded()
			return w.flushBufferedResponse()
		}
	}
}

// checkVersion reports a GET Inertia asset version mismatch.
func checkVersion(req *http.Request, version string) bool {
	return req.Header.Get(HeaderXInertia) != "" &&
		req.Method == http.MethodGet &&
		req.Header.Get(HeaderXInertiaVersion) != version
}

// onVersionChange replaces the handler response with an Inertia location visit.
// Mirrors inertia-laravel Middleware::onVersionChange.
func onVersionChange(c *echo.Context, i *Inertia, w *ResponseWriterWrapper, version string) error {
	if i.reflash != nil {
		if err := i.reflash(c); err != nil {
			return err
		}
	}
	i.sendClearHistoryCookieIfNeeded()
	i.sendPreserveFragmentCookieIfNeeded()

	w.Header().Del(echo.HeaderContentType)
	w.Header().Del(echo.HeaderContentLength)
	w.resetBufferedResponse()
	w.Header().Set(HeaderXInertiaVersion, version)
	return i.Location(c.Request().URL.String())
}

// onEmptyResponse redirects back when an Inertia handler returns an empty 200.
// Mirrors inertia-laravel Middleware::onEmptyResponse (Redirect::back).
func onEmptyResponse(c *echo.Context, w *ResponseWriterWrapper) error {
	target := c.Request().Referer()
	if target == "" {
		target = "/"
	}
	w.resetBufferedResponse()
	return c.Redirect(http.StatusFound, target)
}

// changeRedirectCode changes the status code during redirects, ensuring they are made as
// GET requests, preventing "MethodNotAllowedHttpException" errors.
// Corresponding specification: https://inertiajs.com/docs/v3/the-basics/redirects
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

		return func(c *echo.Context) error {
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
