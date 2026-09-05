package inertia

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

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

func TestWrapHTTPErrorHandlerTransformsRedirect(t *testing.T) {
	e := echo.New()
	e.Use(MiddlewareWithConfig(MiddlewareConfig{VersionFunc: func() string { return "v3" }}))
	e.HTTPErrorHandler = WrapHTTPErrorHandler(func(c *echo.Context, _ error) {
		_ = c.Redirect(http.StatusFound, "/errors#server")
	})
	e.GET("/", func(*echo.Context) error { return errors.New("failed") })
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	req.Header.Set(HeaderXInertiaVersion, "v3")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	if rec.Code != http.StatusConflict || rec.Header().Get(HeaderXInertiaRedirect) != "/errors#server" {
		t.Fatalf("status=%d headers=%v body=%q", rec.Code, rec.Header(), rec.Body.String())
	}
}

func TestHTTPErrorHandlerWithConfigWithoutMiddlewareContext(t *testing.T) {
	e := echo.New()
	shareCalls := 0
	var page *Page
	e.HTTPErrorHandler = HTTPErrorHandlerWithConfig(ErrorHandlerConfig{
		Middleware: MiddlewareConfig{
			Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
				page = ctx.Page
				return nil
			}),
			VersionFunc: func() string { return "v3" },
			Share: func(*echo.Context) (map[string]any, error) {
				shareCalls++
				return map[string]any{"shared": true}, nil
			},
		},
		Component: "Errors/Show",
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/missing", nil))
	if rec.Code != http.StatusNotFound || page == nil || page.Component != "Errors/Show" {
		t.Fatalf("status=%d page=%#v", rec.Code, page)
	}
	if shareCalls != 0 {
		t.Fatalf("shared props must not be reloaded without opt-in: %d", shareCalls)
	}
	if page.Props["status"] != http.StatusNotFound {
		t.Fatalf("props=%v", page.Props)
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
