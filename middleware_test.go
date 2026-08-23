package inertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

const testMiddlewareVersion = "test-version"

func testMiddlewareWithVersion() echo.MiddlewareFunc {
	return MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func() string {
			return testMiddlewareVersion
		},
	})
}

func testNewInertiaRequestContext(method string) (echo.Context, *httptest.ResponseRecorder) {
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
	h := testMiddlewareWithVersion()(func(c echo.Context) error {
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
	h := testMiddlewareWithVersion()(func(c echo.Context) error {
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
			h := testMiddlewareWithVersion()(func(c echo.Context) error {
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
			originalWriter := c.Response().Writer
			h := testMiddlewareWithVersion()(func(c echo.Context) error {
				return tt.err
			})

			err := h(c)
			if !errors.Is(err, tt.err) {
				t.Fatalf("expected error %v, got %v", tt.err, err)
			}
			if c.Response().Writer != originalWriter {
				t.Error("expected the original response writer to be restored")
			}
		})
	}
}
