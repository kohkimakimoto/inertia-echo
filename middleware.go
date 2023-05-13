package inertia

import (
	"errors"
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
}

type SharedDataFunc func(c echo.Context) (map[string]interface{}, error)

var DefaultMiddlewareConfig = MiddlewareConfig{
	Skipper:  middleware.DefaultSkipper,
	RootView: "app.html",
}

func defaultVersionFunc() VersionFunc {
	var v string

	// It is for Google App Engine.
	// see https://cloud.google.com/appengine/docs/standard/go/runtime#environment_variables
	if v = os.Getenv("GAE_VERSION"); v == "" {
		// The fallback version value that imitates the default GAE version format.
		// It assumes to be used for development.
		v = time.Now().Format("20060102t150405")
	}

	return func() string {
		return v
	}
}

func Middleware() echo.MiddlewareFunc {
	return MiddlewareWithConfig(DefaultMiddlewareConfig)
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
		config.VersionFunc = defaultVersionFunc()
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			if config.Skipper(c) {
				return next(c)
			}

			var sharedProps map[string]interface{}
			if config.Share != nil {
				ret, err := config.Share(c)
				if err != nil {
					return err
				}
				sharedProps = ret
			} else {
				sharedProps = map[string]interface{}{}
			}

			// Create an Inertia instance.
			in := New(c, config.RootView, sharedProps, config.VersionFunc)
			c.Set(key, in)

			req := c.Request()
			res := c.Response()

			if req.Header.Get(HeaderXInertia) == "" {
				// Not inertial request
				return next(c)
			}

			// In the event that the assets change, initiate a
			// client-side location visit to force an update.
			// see https://inertiajs.com/the-protocol#asset-versioning
			if checkVersion(req, in.Version()) {
				return in.Location(req.URL.Path)
			}

			// Wrap the http response writer for modify the response headers after handler execution.
			w := NewResponseWriterWrapper(res.Writer)
			res.Writer = w
			defer func(w *ResponseWriterWrapper) {
				// send buffered header and restore the original response writer
				w.FlushHeader()
				res.Writer = w.ResponseWriter
			}(w)

			if err = next(c); err != nil {
				return err
			}

			changeRedirectCode(req, res)

			return nil
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

var (
	ErrNotFound = errors.New("context does not have 'Inertia'")
)

func Get(c echo.Context) (*Inertia, error) {
	in, ok := c.Get(key).(*Inertia)
	if !ok {
		return nil, ErrNotFound
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
