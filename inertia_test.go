package inertia

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRenderPreservesVaryHeader(t *testing.T) {
	const version = "test-version"

	tests := []struct {
		name           string
		inertiaRequest bool
		initialVary    []string
		want           []string
	}{
		{
			name: "response without existing Vary header",
			want: []string{"x-inertia"},
		},
		{
			name:        "HTML response",
			initialVary: []string{"Cookie, Accept-Encoding", "Origin"},
			want:        []string{"accept-encoding", "cookie", "origin", "x-inertia"},
		},
		{
			name:           "Inertia response with X-Inertia already present",
			inertiaRequest: true,
			initialVary:    []string{"Cookie, Accept-Encoding", "Origin, x-inertia"},
			want:           []string{"accept-encoding", "cookie", "origin", "x-inertia"},
		},
		{
			name:        "wildcard",
			initialVary: []string{"*"},
			want:        []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.inertiaRequest {
				req.Header.Set(HeaderXInertia, "true")
				req.Header.Set(HeaderXInertiaVersion, version)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			m := MiddlewareWithConfig(MiddlewareConfig{
				VersionFunc: func(*echo.Context) string {
					return version
				},
				Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
					return nil
				}),
			})
			h := m(func(c *echo.Context) error {
				for _, value := range tt.initialVary {
					c.Response().Header().Add(echo.HeaderVary, value)
				}
				return Render(c, http.StatusOK, "Home", map[string]any{})
			})

			if err := h(c); err != nil {
				t.Fatal(err)
			}

			got := testVaryHeaderTokens(rec.Header())
			if !slices.Equal(got, tt.want) {
				t.Fatalf("unexpected Vary header tokens: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderPageURLDefaultAndResolver(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		urlResolver URLResolverFunc
		wantURL     string
	}{
		{
			name:    "default uses request URL string",
			path:    "/users?page=2",
			wantURL: "/users?page=2",
		},
		{
			name: "custom resolver",
			path: "/users?page=2",
			urlResolver: func(c *echo.Context) string {
				return "/app" + c.Request().URL.String()
			},
			wantURL: "/app/users?page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var page *Page
			e.Use(MiddlewareWithConfig(MiddlewareConfig{
				VersionFunc: func(*echo.Context) string { return "v3" },
				URLResolver: tt.urlResolver,
				Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
					page = ctx.Page
					return nil
				}),
			}))
			e.GET("/users", func(c *echo.Context) error {
				return Render(c, http.StatusOK, "Users", nil)
			})
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusOK || page == nil {
				t.Fatalf("status=%d page=%#v", rec.Code, page)
			}
			if page.URL != tt.wantURL {
				t.Fatalf("URL=%q, want %q", page.URL, tt.wantURL)
			}
		})
	}
}

func TestSetURLResolver(t *testing.T) {
	e := echo.New()
	var page *Page
	e.Use(MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func(*echo.Context) string { return "v3" },
		Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
			page = ctx.Page
			return nil
		}),
	}))
	e.GET("/", func(c *echo.Context) error {
		SetURLResolver(c, func(*echo.Context) string { return "/custom" })
		return Render(c, http.StatusOK, "Home", nil)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || page == nil {
		t.Fatalf("status=%d page=%#v", rec.Code, page)
	}
	if page.URL != "/custom" {
		t.Fatalf("URL=%q, want /custom", page.URL)
	}
}

func testVaryHeaderTokens(header http.Header) []string {
	var tokens []string
	for _, value := range header.Values(echo.HeaderVary) {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				tokens = append(tokens, strings.ToLower(token))
			}
		}
	}
	slices.Sort(tokens)
	return tokens
}
