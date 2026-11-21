package nodejs_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/kohkimakimoto/inertia-echo/v5/ssr/nodejs"
	"github.com/labstack/echo/v5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newInertia(t *testing.T, req *http.Request, mutate func(*inertia.Inertia)) *inertia.Inertia {
	t.Helper()
	e := echo.New()
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: inertia.NewHTMLRenderer(),
	}))
	var got *inertia.Inertia
	path := req.URL.Path
	if path == "" {
		path = "/"
	}
	e.GET(path, func(c *echo.Context) error {
		got = inertia.MustGet(c)
		if mutate != nil {
			mutate(got)
		}
		return c.NoContent(http.StatusNoContent)
	})
	e.ServeHTTP(httptest.NewRecorder(), req)
	if got == nil {
		t.Fatal("expected Inertia in context")
	}
	return got
}

func TestGateway_Dispatch(t *testing.T) {
	tests := []struct {
		name          string
		responseBody  string
		statusCode    int
		want          *inertia.SsrResponse
		errorContains string
	}{
		{
			name:         "successful render",
			responseBody: `{"head":["<title>Test Page</title>"],"body":"<div>Test Content</div>"}`,
			statusCode:   http.StatusOK,
			want: &inertia.SsrResponse{
				Head: []string{"<title>Test Page</title>"},
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
			name:          "invalid JSON",
			responseBody:  `{"head":`,
			statusCode:    http.StatusOK,
			errorContains: "failed to decode response",
		},
		{
			name:          "server error",
			statusCode:    http.StatusInternalServerError,
			errorContains: "status code is not 200",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gateway := &nodejs.Gateway{
				Enabled: true,
				URL:     "http://ssr.test",
				HttpClient: &http.Client{
					Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
						if req.Method != http.MethodPost {
							t.Errorf("expected POST, got %s", req.Method)
						}
						if !strings.HasSuffix(req.URL.Path, "/render") {
							t.Errorf("expected /render path, got %s", req.URL.Path)
						}
						var page inertia.Page
						if err := json.NewDecoder(req.Body).Decode(&page); err != nil {
							t.Fatalf("decode page: %v", err)
						}
						if page.Component != "TestComponent" {
							t.Errorf("expected component TestComponent, got %q", page.Component)
						}
						body := tt.responseBody
						if body == "" {
							body = "{}"
						}
						return &http.Response{
							StatusCode: tt.statusCode,
							Header:     make(http.Header),
							Body:       io.NopCloser(strings.NewReader(body)),
						}, nil
					}),
				},
			}
			ctx := &inertia.RenderContext{
				Page: &inertia.Page{
					Component: "TestComponent",
					Props:     map[string]any{"key": "value"},
					URL:       "/test",
					Version:   "1.0.0",
				},
			}
			got, err := gateway.Dispatch(ctx)
			if tt.errorContains != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errorContains) {
					t.Fatalf("error %v, want containing %q", err, tt.errorContains)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.want == nil {
				if got != nil {
					t.Fatalf("want nil, got %#v", got)
				}
				return
			}
			if got == nil || got.Body != tt.want.Body || strings.Join(got.Head, "\n") != strings.Join(tt.want.Head, "\n") {
				t.Fatalf("got %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestGateway_Dispatch_EndpointSelection(t *testing.T) {
	tests := []struct {
		name         string
		gateway      *nodejs.Gateway
		wantURLPath  string
		wantHostHint string
	}{
		{
			name: "production URL",
			gateway: &nodejs.Gateway{
				Enabled: true,
				URL:     "http://ssr.prod",
			},
			wantURLPath:  "/render",
			wantHostHint: "ssr.prod",
		},
		{
			name: "hot URL",
			gateway: &nodejs.Gateway{
				Enabled: true,
				URL:     "http://ssr.prod",
				HotURL:  "http://vite.dev",
				UseHot:  true,
			},
			wantURLPath:  "/__inertia_ssr",
			wantHostHint: "vite.dev",
		},
		{
			name: "explicit endpoint wins",
			gateway: &nodejs.Gateway{
				Enabled:  true,
				URL:      "http://ssr.prod",
				HotURL:   "http://vite.dev",
				UseHot:   true,
				Endpoint: "http://custom.test/render-me",
			},
			wantURLPath:  "/render-me",
			wantHostHint: "custom.test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saw *http.Request
			tt.gateway.HttpClient = &http.Client{
				Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
					saw = req
					return &http.Response{
						StatusCode: http.StatusOK,
						Header:     make(http.Header),
						Body:       io.NopCloser(strings.NewReader(`{"head":[],"body":"<div/>"}`)),
					}, nil
				}),
			}
			_, err := tt.gateway.Dispatch(&inertia.RenderContext{
				Page: &inertia.Page{Component: "X", Props: map[string]any{}},
			})
			if err != nil {
				t.Fatal(err)
			}
			if saw == nil {
				t.Fatal("no request")
			}
			if saw.URL.Path != tt.wantURLPath {
				t.Fatalf("path %q, want %q", saw.URL.Path, tt.wantURLPath)
			}
			if !strings.Contains(saw.URL.Host, tt.wantHostHint) {
				t.Fatalf("host %q, want containing %q", saw.URL.Host, tt.wantHostHint)
			}
		})
	}
}

func TestNewGateway(t *testing.T) {
	g := nodejs.NewGateway()
	if !g.Enabled || g.URL != "http://127.0.0.1:13714" || g.HttpClient == nil {
		t.Fatalf("unexpected defaults: %#v", g)
	}
}

func TestGateway_Dispatch_NetworkError(t *testing.T) {
	g := &nodejs.Gateway{Enabled: true, URL: "http://invalid-host:99999", HttpClient: &http.Client{}}
	_, err := g.Dispatch(&inertia.RenderContext{Page: &inertia.Page{Component: "X", Props: map[string]any{}}})
	if err == nil {
		t.Fatal("expected network error")
	}
}

func TestGateway_Dispatch_PropagatesRequestContext(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext)
	in := newInertia(t, req, nil)

	g := &nodejs.Gateway{
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
					}, nil
				}
			}),
		},
	}
	_, err := g.Dispatch(&inertia.RenderContext{
		Inertia: in,
		Page:    &inertia.Page{Component: "X", Props: map[string]any{}},
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

func TestGateway_Dispatch_Skips(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("SSR server should not be called")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	t.Run("Enabled false", func(t *testing.T) {
		g := &nodejs.Gateway{Enabled: false, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(&inertia.RenderContext{Page: &inertia.Page{Props: map[string]any{}}})
		if err != nil || got != nil {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})

	t.Run("DisableSsr on Inertia", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		in := newInertia(t, req, func(i *inertia.Inertia) { i.DisableSsr() })
		g := &nodejs.Gateway{Enabled: true, URL: server.URL, HttpClient: &http.Client{}}
		got, err := g.Dispatch(&inertia.RenderContext{Inertia: in, Page: &inertia.Page{Props: map[string]any{}}})
		if err != nil || got != nil {
			t.Fatalf("got (%v, %v)", got, err)
		}
	})
}

func TestGateway_EnsureBundleExists(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "ssr.js")
	if err := os.WriteFile(bundle, []byte("//"), 0o644); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		_, _ = w.Write([]byte(`{"head":[],"body":"<div/>"}`))
	}))
	defer server.Close()

	t.Run("missing bundle skips", func(t *testing.T) {
		called = false
		g := &nodejs.Gateway{
			Enabled:            true,
			URL:                server.URL,
			EnsureBundleExists: true,
			BundlePaths:        []string{filepath.Join(dir, "missing.js")},
			HttpClient:         &http.Client{},
		}
		got, err := g.Dispatch(&inertia.RenderContext{Page: &inertia.Page{Props: map[string]any{}}})
		if err != nil || got != nil || called {
			t.Fatalf("got (%v,%v) called=%v", got, err, called)
		}
	})

	t.Run("existing bundle renders", func(t *testing.T) {
		called = false
		g := &nodejs.Gateway{
			Enabled:            true,
			URL:                server.URL,
			EnsureBundleExists: true,
			BundlePaths:        []string{bundle},
			HttpClient:         &http.Client{},
		}
		got, err := g.Dispatch(&inertia.RenderContext{Page: &inertia.Page{Props: map[string]any{}}})
		if err != nil || got == nil || !called {
			t.Fatalf("got (%v,%v) called=%v", got, err, called)
		}
	})
}

func TestGateway_IsHealthy(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("path %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	g := &nodejs.Gateway{URL: server.URL, HttpClient: &http.Client{}}
	if !g.IsHealthy(context.Background()) {
		t.Fatal("expected healthy")
	}
}
