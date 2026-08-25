package inertia

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

const testMiddlewareVersion = "test-version"

func testMiddlewareWithVersion() echo.MiddlewareFunc {
	return MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func() string {
			return testMiddlewareVersion
		},
	})
}

func testNewInertiaRequestContext(method string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, testMiddlewareVersion)
	rec := httptest.NewRecorder()
	return e.NewContext(req, rec), rec
}

func TestMiddleware_AssetVersionMismatchPreservesRequestURL(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users?page=2&filter=active", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, "stale-version")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	h := testMiddlewareWithVersion()(func(c *echo.Context) error {
		t.Fatal("expected asset version mismatch to skip the handler")
		return nil
	})

	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusConflict {
		t.Errorf("expected status code %d, got %d", http.StatusConflict, rec.Code)
	}
	if location := rec.Header().Get(HeaderXInertiaLocation); location != "/users?page=2&filter=active" {
		t.Errorf("expected Inertia location %q, got %q", "/users?page=2&filter=active", location)
	}
}

func TestMiddleware_NetHTTPRedirect(t *testing.T) {
	c, rec := testNewInertiaRequestContext(http.MethodGet)
	h := testMiddlewareWithVersion()(echo.WrapHandler(
		http.RedirectHandler("/next", http.StatusFound),
	))

	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/next" {
		t.Errorf("expected Location header %q, got %q", "/next", location)
	}
	if count := strings.Count(rec.Body.String(), "/next"); count != 1 {
		t.Errorf("expected redirect body to contain the target once, got %d occurrences in %q", count, rec.Body.String())
	}
}

func TestMiddleware_EchoRedirect(t *testing.T) {
	c, rec := testNewInertiaRequestContext(http.MethodGet)
	h := testMiddlewareWithVersion()(func(c *echo.Context) error {
		return c.Redirect(http.StatusFound, "/next")
	})

	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, rec.Code)
	}
	if location := rec.Header().Get("Location"); location != "/next" {
		t.Errorf("expected Location header %q, got %q", "/next", location)
	}
}

func TestMiddleware_ChangesRedirectStatusToSeeOther(t *testing.T) {
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			c, rec := testNewInertiaRequestContext(method)
			h := testMiddlewareWithVersion()(func(c *echo.Context) error {
				return c.Redirect(http.StatusFound, "/next")
			})

			if err := h(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status code %d, got %d", http.StatusSeeOther, rec.Code)
			}
		})
	}
}

func TestMiddleware_RestoresResponseWriter(t *testing.T) {
	handlerErr := errors.New("handler failed")
	tests := []struct {
		name string
		err  error
	}{
		{name: "successful handler"},
		{name: "handler error", err: handlerErr},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := testNewInertiaRequestContext(http.MethodGet)
			originalWriter := c.Response()
			h := testMiddlewareWithVersion()(func(c *echo.Context) error {
				if c.Response() == originalWriter {
					t.Error("expected the response writer to be wrapped while the handler runs")
				}
				return tt.err
			})

			err := h(c)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if c.Response() != originalWriter {
				t.Error("expected the original response writer to be restored")
			}
		})
	}
}

func TestMiddleware_RestoresResponseWriterBeforeErrorHandler(t *testing.T) {
	e := echo.New()
	e.Use(testMiddlewareWithVersion())
	e.GET("/", func(c *echo.Context) error {
		return echo.ErrNotFound
	})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, testMiddlewareVersion)
	rec := httptest.NewRecorder()

	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected status code %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestMiddleware_ResponseWriterCanBeUnwrapped(t *testing.T) {
	c, _ := testNewInertiaRequestContext(http.MethodGet)
	want, err := echo.UnwrapResponse(c.Response())
	if err != nil {
		t.Fatal(err)
	}
	h := testMiddlewareWithVersion()(func(c *echo.Context) error {
		got, err := echo.UnwrapResponse(c.Response())
		if err != nil {
			t.Fatalf("expected the Inertia response writer to unwrap to Echo's response: %v", err)
		}
		if got != want {
			t.Error("expected the wrapper to unwrap to the original Echo response")
		}
		return nil
	})

	if err := h(c); err != nil {
		t.Fatal(err)
	}
}

func TestMiddleware_ComposesWithGzipMiddleware(t *testing.T) {
	tests := []struct {
		name        string
		middlewares []echo.MiddlewareFunc
	}{
		{
			name:        "Inertia outside Gzip",
			middlewares: []echo.MiddlewareFunc{testMiddlewareWithVersion(), middleware.Gzip()},
		},
		{
			name:        "Gzip outside Inertia",
			middlewares: []echo.MiddlewareFunc{middleware.Gzip(), testMiddlewareWithVersion()},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			e.Use(tt.middlewares...)
			e.PUT("/", func(c *echo.Context) error {
				if _, err := echo.UnwrapResponse(c.Response()); err != nil {
					t.Fatalf("expected the middleware chain to unwrap to Echo's response: %v", err)
				}
				c.Response().Header().Set(echo.HeaderLocation, "/next")
				return c.String(http.StatusFound, "redirect body")
			})

			req := httptest.NewRequest(http.MethodPut, "/", nil)
			req.Header.Set(HeaderXInertia, "true")
			req.Header.Set(HeaderXInertiaVersion, testMiddlewareVersion)
			req.Header.Set(echo.HeaderAcceptEncoding, "gzip")
			rec := httptest.NewRecorder()

			e.ServeHTTP(rec, req)

			if rec.Code != http.StatusSeeOther {
				t.Errorf("expected status code %d, got %d", http.StatusSeeOther, rec.Code)
			}
			if location := rec.Header().Get(echo.HeaderLocation); location != "/next" {
				t.Errorf("expected Location header %q, got %q", "/next", location)
			}
			if encoding := rec.Header().Get(echo.HeaderContentEncoding); encoding != "gzip" {
				t.Fatalf("expected Content-Encoding %q, got %q", "gzip", encoding)
			}

			reader, err := gzip.NewReader(rec.Body)
			if err != nil {
				t.Fatal(err)
			}
			defer reader.Close()
			body, err := io.ReadAll(reader)
			if err != nil {
				t.Fatal(err)
			}
			if string(body) != "redirect body" {
				t.Errorf("expected decompressed body %q, got %q", "redirect body", body)
			}
		})
	}
}
