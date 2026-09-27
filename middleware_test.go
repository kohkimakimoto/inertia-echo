package inertia

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"slices"
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

func TestMiddlewareAssetVersionMismatchV3Headers(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/users?page=2", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, "old")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	handlerCalls := 0
	err := MiddlewareWithConfig(MiddlewareConfig{VersionFunc: func() string { return "new" }})(func(*echo.Context) error {
		handlerCalls++
		return nil
	})(c)
	if err != nil {
		t.Fatal(err)
	}
	if handlerCalls != 0 || rec.Code != http.StatusConflict {
		t.Fatalf("handlerCalls=%d status=%d", handlerCalls, rec.Code)
	}
	if got := rec.Header().Get(HeaderXInertiaLocation); got != "/users?page=2" {
		t.Fatalf("location=%q", got)
	}
	if got := rec.Header().Get(HeaderXInertiaVersion); got != "new" {
		t.Fatalf("version=%q", got)
	}
	if got := testVaryHeaderTokens(rec.Header()); !slices.Contains(got, "x-inertia") {
		t.Fatalf("Vary=%v", got)
	}
}

func TestMiddlewareAssetVersionMismatchResolvesVersionOnce(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, "old")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	calls := 0
	h := MiddlewareWithConfig(MiddlewareConfig{VersionFunc: func() string {
		calls++
		return "new"
	}})(func(*echo.Context) error {
		t.Fatal("handler must not run on a version mismatch")
		return nil
	})
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("VersionFunc calls=%d, want 1", calls)
	}
	if got := rec.Header().Get(HeaderXInertiaVersion); got != "new" {
		t.Fatalf("version=%q", got)
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

func TestMiddlewareFragmentRedirectContract(t *testing.T) {
	statuses := []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect}
	for _, status := range statuses {
		t.Run(http.StatusText(status), func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.Header.Set(HeaderXInertia, "true")
			req.Header.Set(HeaderXInertiaVersion, "v3")
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			h := MiddlewareWithConfig(MiddlewareConfig{VersionFunc: func() string { return "v3" }})(func(c *echo.Context) error {
				http.Redirect(c.Response(), c.Request(), "/target#details", status)
				return nil
			})
			if err := h(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusConflict {
				t.Fatalf("status=%d", rec.Code)
			}
			if got := rec.Header().Get(HeaderXInertiaRedirect); got != "/target#details" {
				t.Fatalf("redirect=%q", got)
			}
			if got := rec.Header().Get("Location"); got != "" {
				t.Fatalf("legacy Location remained: %q", got)
			}
			if rec.Body.Len() != 0 {
				t.Fatalf("legacy redirect body remained: %q", rec.Body.String())
			}
		})
	}
}

func TestMiddlewareFragmentRedirectExceptions(t *testing.T) {
	tests := []struct {
		name     string
		inertia  bool
		prefetch bool
	}{
		{name: "normal request"},
		{name: "prefetch", inertia: true, prefetch: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.inertia {
				req.Header.Set(HeaderXInertia, "true")
				req.Header.Set(HeaderXInertiaVersion, "v3")
			}
			if tt.prefetch {
				req.Header.Set("Purpose", "prefetch")
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)
			h := MiddlewareWithConfig(MiddlewareConfig{VersionFunc: func() string { return "v3" }})(func(c *echo.Context) error {
				return c.Redirect(http.StatusFound, "/target#details")
			})
			if err := h(c); err != nil {
				t.Fatal(err)
			}
			if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/target#details" {
				t.Fatalf("status=%d headers=%v", rec.Code, rec.Header())
			}
			if rec.Header().Get(HeaderXInertiaRedirect) != "" {
				t.Fatal("fragment redirect conversion must be skipped")
			}
		})
	}
}

func TestMiddlewareFinalizesOneShotStateBeforeRedirect(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, "v3")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	reflashCalls := 0
	h := MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func() string { return "v3" },
		Reflash: func(*echo.Context) error {
			reflashCalls++
			return nil
		},
	})(func(c *echo.Context) error {
		i := MustGet(c)
		i.ClearHistory()
		i.PreserveFragment()
		return c.Redirect(http.StatusFound, "/next")
	})
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if reflashCalls != 1 {
		t.Fatalf("reflashCalls=%d", reflashCalls)
	}
	cookies := rec.Result().Cookies()
	var names []string
	for _, cookie := range cookies {
		names = append(names, cookie.Name)
	}
	if !slices.Contains(names, "inertia.clear_history") || !slices.Contains(names, "inertia.preserve_fragment") {
		t.Fatalf("one-shot cookies=%v", names)
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

func TestFlashAndRenderWithStatus(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	var page *Page
	renderer := testNewMockRenderer(t, func(ctx *RenderContext) error {
		page = ctx.Page
		return nil
	})
	h := MiddlewareWithConfig(MiddlewareConfig{
		Renderer: renderer,
		FlashData: func(*echo.Context) (map[string]any, error) {
			return map[string]any{"message": "stored", "old": true}, nil
		},
	})(func(c *echo.Context) error {
		Flash(c, map[string]any{"message": "local"})
		return RenderWithStatus(c, http.StatusNotFound, "ErrorPage", nil)
	})
	if err := h(c); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusNotFound || page == nil {
		t.Fatalf("status=%d page=%v", rec.Code, page)
	}
	if page.Flash["message"] != "local" || page.Flash["old"] != true {
		t.Fatalf("flash=%v", page.Flash)
	}
}

func TestMiddlewareShareOnce(t *testing.T) {
	e := echo.New()
	var page *Page
	countriesCalls := 0
	e.Use(MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func() string { return "v3" },
		Share: func(*echo.Context) (map[string]any, error) {
			return map[string]any{"appName": "Demo"}, nil
		},
		ShareOnce: func(*echo.Context) (map[string]any, error) {
			return map[string]any{
				"countries": func() (any, error) {
					countriesCalls++
					return []string{"JP", "US"}, nil
				},
				"features": Once(func() (any, error) {
					return []string{"billing"}, nil
				}).As("feature-flags"),
			}, nil
		},
		Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
			page = ctx.Page
			return nil
		}),
	}))
	e.GET("/", func(c *echo.Context) error {
		return Render(c, "Home", nil)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || page == nil {
		t.Fatalf("status=%d page=%#v", rec.Code, page)
	}
	if page.Props["appName"] != "Demo" {
		t.Fatalf("shared props=%v", page.Props)
	}
	if got, _ := page.Props["countries"].([]string); len(got) != 2 || got[0] != "JP" {
		t.Fatalf("countries=%v", page.Props["countries"])
	}
	if got, _ := page.Props["features"].([]string); len(got) != 1 || got[0] != "billing" {
		t.Fatalf("features=%v", page.Props["features"])
	}
	if countriesCalls != 1 {
		t.Fatalf("countriesCalls=%d", countriesCalls)
	}
	if page.OnceProps["countries"].Prop != "countries" {
		t.Fatalf("onceProps=%v", page.OnceProps)
	}
	if meta, ok := page.OnceProps["feature-flags"]; !ok || meta.Prop != "features" {
		t.Fatalf("onceProps=%v", page.OnceProps)
	}
}

func TestPageOmitsFalseHistoryFlags(t *testing.T) {
	encoded, err := json.Marshal(Page{Props: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	got := string(encoded)
	if strings.Contains(got, "encryptHistory") || strings.Contains(got, "clearHistory") || strings.Contains(got, "preserveFragment") {
		t.Fatalf("false flags must be omitted: %s", got)
	}
}
