package inertia

import (
	"context"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSsrHTTPGateway_Dispatch(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		statusCode    int
		want          *SsrResponse
		errorContains string
	}{
		{
			name:         "successful render",
			responseBody: `{"head":["<title>Test Page</title>","<meta name=\"description\" content=\"test\">"],"body":"<div>Test Content</div>"}`,
			statusCode:   http.StatusOK,
			want: &SsrResponse{
				Head: []string{"<title>Test Page</title>", "<meta name=\"description\" content=\"test\">"},
				Body: "<div>Test Content</div>",
			},
		},
		{
			name:         "null response skips SSR",
			responseBody: "null",
			statusCode:   http.StatusOK,
			want:         nil,
		},
		{
			name:          "missing body",
			responseBody:  `{"head":[]}`,
			statusCode:    http.StatusOK,
			errorContains: "response body is missing or null",
		},
		{
			name:          "null body",
			responseBody:  `{"head":[],"body":null}`,
			statusCode:    http.StatusOK,
			errorContains: "response body is missing or null",
		},
		{
			name:          "body has invalid type",
			responseBody:  `{"head":[],"body":42}`,
			statusCode:    http.StatusOK,
			errorContains: "failed to decode response",
		},
		{
			name:          "invalid JSON",
			responseBody:  `{"head":`,
			statusCode:    http.StatusOK,
			errorContains: "failed to decode response",
		},
		{
			name:          "trailing invalid JSON",
			responseBody:  `{"head":[],"body":"<div>SSR</div>"} trailing`,
			statusCode:    http.StatusOK,
			errorContains: "failed to decode response",
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			errorContains: "status code is not 200: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPost {
					t.Errorf("expected POST request, got %s", r.Method)
				}
				if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
					t.Errorf("expected Content-Type application/json, got %q", contentType)
				}

				var page Page
				if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
					t.Errorf("failed to decode request body: %v", err)
				}
				if page.Component != "TestComponent" {
					t.Errorf("expected direct Page payload, got component %q", page.Component)
				}

				w.WriteHeader(tt.statusCode)
				if tt.responseBody != "" {
					_, _ = io.WriteString(w, tt.responseBody)
				}
			}))
			defer server.Close()

			gateway := &SsrHTTPGateway{
				Enabled:    true,
				URL:        server.URL,
				HttpClient: &http.Client{},
			}

			ctx := &RenderContext{
				Page: &Page{
					Component: "TestComponent",
					Props:     map[string]any{"key": "value"},
					URL:       "/test",
					Version:   "1.0.0",
				},
			}

			got, err := gateway.Dispatch(ctx)
			if tt.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("expected error containing %q, got %v", tt.errorContains, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("expected nil SSR response, got %#v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil SSR response")
			}
			if !testDeepEqual(t, got.Head, tt.want.Head) {
				t.Errorf("expected head %v, got %v", tt.want.Head, got.Head)
			}
			if got.Body != tt.want.Body {
				t.Errorf("expected body %q, got %q", tt.want.Body, got.Body)
			}
		})
	}
}

func TestSsrHTTPGateway_Dispatch_EndpointSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"head":[],"body":"<div>SSR</div>"}`)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		url      string
		hotURL   string
		useHot   bool
		endpoint string
		wantPath string
	}{
		{
			name:     "production URL",
			url:      server.URL,
			wantPath: "/render",
		},
		{
			name:     "production URL with trailing slash",
			url:      server.URL + "/",
			wantPath: "/render",
		},
		{
			name:     "hot URL",
			url:      "http://invalid.example",
			hotURL:   server.URL,
			useHot:   true,
			wantPath: "/__inertia_ssr",
		},
		{
			name:     "Endpoint takes precedence",
			url:      "http://invalid.example",
			endpoint: server.URL + "/__inertia_ssr",
			wantPath: "/__inertia_ssr",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					if req.URL.Path != tt.wantPath {
						t.Errorf("expected request path %q, got %q", tt.wantPath, req.URL.Path)
					}
					return http.DefaultTransport.RoundTrip(req)
				}),
			}
			gateway := &SsrHTTPGateway{
				Enabled:    true,
				URL:        tt.url,
				HotURL:     tt.hotURL,
				UseHot:     tt.useHot,
				Endpoint:   tt.endpoint,
				HttpClient: client,
			}

			response, err := gateway.Dispatch(&RenderContext{Page: &Page{
				Component: "TestComponent",
				Props:     map[string]any{},
				URL:       "/test",
				Version:   "1.0.0",
			}})
			if err != nil {
				t.Fatal(err)
			}
			if response == nil || response.Body != "<div>SSR</div>" {
				t.Fatalf("unexpected SSR response: %#v", response)
			}
		})
	}
}

func TestSsrHTTPGateway_Dispatch_UsesDefaultClientWhenNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"head":[],"body":"<div>SSR</div>"}`)
	}))
	defer server.Close()

	gateway := &SsrHTTPGateway{Enabled: true, URL: server.URL}
	response, err := gateway.Dispatch(&RenderContext{Page: &Page{Props: map[string]any{}}})
	if err != nil {
		t.Fatal(err)
	}
	if response == nil || response.Body != "<div>SSR</div>" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestSsrResponse_HeadHTML(t *testing.T) {
	response := &SsrResponse{
		Head: []string{
			"<title>Test Page</title>",
			"<meta name=\"description\" content=\"test\">",
			"<link rel=\"stylesheet\" href=\"/style.css\">",
		},
	}

	result := response.HeadHTML()
	expected := "<title>Test Page</title>\n<meta name=\"description\" content=\"test\">\n<link rel=\"stylesheet\" href=\"/style.css\">"

	if string(result) != expected {
		t.Errorf("Expected HeadHTML to return '%s', got '%s'", expected, string(result))
	}
}

func TestSsrResponse_BodyHTML(t *testing.T) {
	response := &SsrResponse{
		Body: "<div><h1>Hello World</h1><p>This is a test</p></div>",
	}

	result := response.BodyHTML()
	expected := "<div><h1>Hello World</h1><p>This is a test</p></div>"

	if string(result) != expected {
		t.Errorf("Expected BodyHTML to return '%s', got '%s'", expected, string(result))
	}
}

func TestNewSsrHTTPGateway(t *testing.T) {
	gateway := NewSsrHTTPGateway()

	if gateway == nil {
		t.Error("Expected non-nil gateway")
		return
	}
	if !gateway.Enabled {
		t.Error("Expected Enabled true by default")
	}
	if gateway.URL != "http://127.0.0.1:13714" {
		t.Errorf("Expected default URL 'http://127.0.0.1:13714', got '%s'", gateway.URL)
	}
	if gateway.Endpoint != "" {
		t.Errorf("expected default Endpoint to be empty, got %q", gateway.Endpoint)
	}
	if gateway.HttpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}
	if gateway.HttpClient.Timeout != 0 {
		t.Errorf("Expected default HTTP client timeout to be disabled, got %s", gateway.HttpClient.Timeout)
	}
	if gateway.FallbackOnError {
		t.Error("Expected FallbackOnError false by default")
	}
}

func TestSsrHTTPGateway_Dispatch_NetworkError(t *testing.T) {
	gateway := &SsrHTTPGateway{
		Enabled:    true,
		URL:        "http://invalid-host:99999",
		HttpClient: &http.Client{},
	}

	ctx := &RenderContext{
		Page: &Page{
			Component: "TestComponent",
			Props:     map[string]any{"key": "value"},
			URL:       "/test",
			Version:   "1.0.0",
		},
	}

	_, err := gateway.Dispatch(ctx)
	if err == nil {
		t.Error("Expected network error but got none")
	}
}

func TestSsrHTTPGateway_Dispatch_FallbackOnError(t *testing.T) {
	reported := false
	gateway := &SsrHTTPGateway{
		Enabled:         true,
		URL:             "http://invalid-host:99999",
		HttpClient:      &http.Client{},
		FallbackOnError: true,
		ErrorReporter:   func(*RenderContext, error) { reported = true },
	}
	got, err := gateway.Dispatch(&RenderContext{Page: &Page{Props: map[string]any{}}})
	if err != nil {
		t.Fatalf("expected soft fallback, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil response, got %#v", got)
	}
	if !reported {
		t.Fatal("expected ErrorReporter to be called")
	}
}

func TestSsrHTTPGateway_Dispatch_PropagatesRequestContext(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext)
	c := e.NewContext(req, httptest.NewRecorder())
	gateway := &SsrHTTPGateway{
		Enabled: true,
		URL:     "http://ssr.test",
		HttpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				default:
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(`{"head":[],"body":""}`)),
						Request:    req,
					}, nil
				}
			}),
		},
		FallbackOnError: true,
	}
	renderContext := &RenderContext{
		Inertia: &Inertia{echoContext: c},
		Page: &Page{
			Component: "TestComponent",
			Props:     map[string]any{},
			URL:       "/",
			Version:   "1.0.0",
		},
	}

	_, err := gateway.Dispatch(renderContext)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestSsrHTTPGateway_Dispatch_InvalidPageJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(SsrResponse{
			Head: []string{"<title>Test</title>"},
			Body: "<div>Test</div>",
		})
	}))
	defer server.Close()

	gateway := &SsrHTTPGateway{
		Enabled:    true,
		URL:        server.URL,
		HttpClient: &http.Client{},
	}

	ctx := &RenderContext{
		Page: &Page{
			Component: "TestComponent",
			Props:     map[string]any{"invalid": make(chan int)},
			URL:       "/test",
			Version:   "1.0.0",
		},
	}

	_, err := gateway.Dispatch(ctx)
	if err == nil {
		t.Error("Expected JSON marshal error but got none")
	}
	if !strings.Contains(err.Error(), "failed to marshal page json") {
		t.Errorf("Expected error about JSON marshal, got: %v", err)
	}
}

func TestSsrHTTPGateway_ssrIsEnabledControls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		t.Fatal("SSR HTTP must not be called when disabled/skipped")
	}))
	defer server.Close()

	newCtx := func(path string, mutate func(*Inertia)) *RenderContext {
		e := echo.New()
		c := e.NewContext(httptest.NewRequest(http.MethodGet, path, nil), httptest.NewRecorder())
		i := &Inertia{echoContext: c}
		if mutate != nil {
			mutate(i)
		}
		return &RenderContext{Inertia: i, Page: &Page{Props: map[string]any{}}}
	}

	t.Run("Enabled false", func(t *testing.T) {
		g := &SsrHTTPGateway{Enabled: false, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(newCtx("/", nil))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("DisableSsr on Inertia", func(t *testing.T) {
		g := &SsrHTTPGateway{Enabled: true, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(newCtx("/", func(i *Inertia) { i.DisableSsr() }))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("DisableSsrWhen", func(t *testing.T) {
		g := &SsrHTTPGateway{Enabled: true, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(newCtx("/", func(i *Inertia) {
			i.DisableSsrWhen(func(*echo.Context) bool { return true })
		}))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("WithoutSsr path match", func(t *testing.T) {
		g := &SsrHTTPGateway{Enabled: true, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(newCtx("/admin/users", func(i *Inertia) {
			i.WithoutSsr("admin/*")
		}))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("base Except path match", func(t *testing.T) {
		g := &SsrHTTPGateway{Enabled: true, URL: server.URL, HttpClient: &http.Client{}, Except: []string{"admin/*"}}
		got, err := g.Dispatch(newCtx("/admin/users", nil))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("base Disable func", func(t *testing.T) {
		g := &SsrHTTPGateway{
			Enabled:    true,
			URL:        server.URL,
			HttpClient: &http.Client{},
			Disable:    func(*echo.Context) bool { return true },
		}
		got, err := g.Dispatch(newCtx("/", nil))
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
}

func TestSsrHTTPGateway_EnsureBundleExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"head":[],"body":"<div>SSR</div>"}`)
	}))
	defer server.Close()

	t.Run("missing bundle skips", func(t *testing.T) {
		g := &SsrHTTPGateway{
			Enabled:            true,
			URL:                server.URL,
			HttpClient:         &http.Client{},
			EnsureBundleExists: true,
			BundlePaths:        []string{filepath.Join(t.TempDir(), "missing.js")},
		}
		got, err := g.Dispatch(&RenderContext{Page: &Page{Props: map[string]any{}}})
		if err != nil || got != nil {
			t.Fatalf("got=%#v err=%v", got, err)
		}
	})
	t.Run("existing bundle dispatches", func(t *testing.T) {
		dir := t.TempDir()
		bundle := filepath.Join(dir, "ssr.js")
		if err := os.WriteFile(bundle, []byte("// ssr"), 0o644); err != nil {
			t.Fatal(err)
		}
		g := &SsrHTTPGateway{
			Enabled:            true,
			URL:                server.URL,
			HttpClient:         &http.Client{},
			EnsureBundleExists: true,
			BundlePaths:        []string{bundle},
		}
		got, err := g.Dispatch(&RenderContext{Page: &Page{Props: map[string]any{}}})
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || got.Body != "<div>SSR</div>" {
			t.Fatalf("got=%#v", got)
		}
	})
}

func TestSsrHTTPGateway_IsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Fatalf("path=%q", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	g := &SsrHTTPGateway{URL: server.URL, HttpClient: &http.Client{}}
	if !g.IsHealthy(context.Background()) {
		t.Fatal("expected healthy")
	}

	g.URL = "http://invalid-host:99999"
	if g.IsHealthy(context.Background()) {
		t.Fatal("expected unhealthy")
	}
}

func TestSsrHTTPGateway_ExceptPaths(t *testing.T) {
	g := NewSsrHTTPGateway()
	g.ExceptPaths("admin/*", "internal")
	if len(g.Except) != 2 || g.Except[0] != "admin/*" || g.Except[1] != "internal" {
		t.Fatalf("Except=%v", g.Except)
	}
}

func TestMatchSSRPath(t *testing.T) {
	tests := []struct {
		path, pattern string
		want          bool
	}{
		{"/admin/users", "admin/*", true},
		{"/admin/users", "/admin/*", true},
		{"/users", "admin/*", false},
		{"/", "/", true},
		{"/exact", "exact", true},
	}
	for _, tt := range tests {
		if got := matchSSRPath(tt.path, tt.pattern); got != tt.want {
			t.Fatalf("matchSSRPath(%q, %q)=%v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}

func TestMiddlewareWithoutSsr(t *testing.T) {
	e := echo.New()
	dispatched := false
	gateway := gatewayFunc(func(ctx *RenderContext) (*SsrResponse, error) {
		dispatched = true
		return &SsrResponse{Body: "<div>ssr</div>"}, nil
	})
	r := NewHTMLRenderer()
	r.SsrGateway = gateway
	r.Templates = mustParseRendererTemplate(t, r, `{{ define "app.html" }}{{ .inertia }}{{ end }}`)

	e.Use(MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func(*echo.Context) string { return "v3" },
		WithoutSsr:  []string{"admin/*"},
		Renderer:    r,
	}))
	e.GET("/admin/users", func(c *echo.Context) error {
		if SsrEnabled(c) {
			t.Fatal("SsrEnabled should be false for WithoutSsr match")
		}
		return Render(c, "Home", nil)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if dispatched {
		t.Fatal("SsrGateway.Dispatch must not run when Inertia.SsrEnabled is false")
	}
	if !strings.Contains(rec.Body.String(), `type="application/json"`) {
		t.Fatalf("expected CSR bootstrap, got %s", rec.Body.String())
	}
}

func TestInertiaSsrEnabled(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/admin/users", nil), httptest.NewRecorder())
	i := &Inertia{echoContext: c}
	if !i.SsrEnabled() {
		t.Fatal("default should allow SSR")
	}
	i.DisableSsr()
	if i.SsrEnabled() {
		t.Fatal("DisableSsr")
	}
	i.ssrDisabled = false
	i.WithoutSsr("admin/*")
	if i.SsrEnabled() {
		t.Fatal("WithoutSsr path match")
	}
	i.withoutSsr = nil
	i.DisableSsrWhen(func(*echo.Context) bool { return true })
	if i.SsrEnabled() {
		t.Fatal("DisableSsrWhen")
	}
}

type gatewayFunc func(*RenderContext) (*SsrResponse, error)

func (f gatewayFunc) Dispatch(ctx *RenderContext) (*SsrResponse, error) { return f(ctx) }

func mustParseRendererTemplate(t *testing.T, r *HTMLRenderer, body string) *template.Template {
	t.Helper()
	tmpl, err := template.New("").Funcs(r.FuncMap()).Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}
