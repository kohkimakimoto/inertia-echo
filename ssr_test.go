package inertia

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestSsrEngineHTTPGateway_Render(t *testing.T) {
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

			engine := &SsrEngineHTTPGateway{
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

			got, err := engine.Render(ctx)
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

func TestSsrEngineHTTPGateway_Render_EndpointSelection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, `{"head":[],"body":"<div>SSR</div>"}`)
	}))
	defer server.Close()

	tests := []struct {
		name     string
		url      string
		endpoint string
		wantPath string
	}{
		{
			name:     "legacy URL",
			url:      server.URL,
			wantPath: "/render",
		},
		{
			name:     "legacy URL with trailing slash",
			url:      server.URL + "/",
			wantPath: "/render",
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
			engine := &SsrEngineHTTPGateway{
				URL:        tt.url,
				Endpoint:   tt.endpoint,
				HttpClient: client,
			}

			response, err := engine.Render(&RenderContext{Page: &Page{
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

func TestSsrEngineHTTPGateway_Render_UsesDefaultClientWhenNil(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"head":[],"body":"<div>SSR</div>"}`)
	}))
	defer server.Close()

	engine := &SsrEngineHTTPGateway{URL: server.URL}
	response, err := engine.Render(&RenderContext{Page: &Page{Props: map[string]any{}}})
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

func TestNewSsrEngineHTTPGateway(t *testing.T) {
	engine := NewSsrEngineHTTPGateway()

	if engine == nil {
		t.Error("Expected non-nil engine")
		return
	}

	if engine.URL != "http://127.0.0.1:13714" {
		t.Errorf("Expected default URL 'http://127.0.0.1:13714', got '%s'", engine.URL)
	}
	if engine.Endpoint != "" {
		t.Errorf("expected default Endpoint to be empty, got %q", engine.Endpoint)
	}

	if engine.HttpClient == nil {
		t.Error("Expected non-nil HTTP client")
	}

	if engine.HttpClient.Timeout != 0 {
		t.Errorf("Expected default HTTP client timeout to be disabled, got %s", engine.HttpClient.Timeout)
	}
}

func TestSsrEngineHTTPGateway_Render_NetworkError(t *testing.T) {
	// Create SSR engine with invalid URL to simulate network error
	engine := &SsrEngineHTTPGateway{
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

	_, err := engine.Render(ctx)
	if err == nil {
		t.Error("Expected network error but got none")
	}
}

func TestSsrEngineHTTPGateway_Render_PropagatesRequestContext(t *testing.T) {
	requestContext, cancel := context.WithCancel(context.Background())
	cancel()

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(requestContext)
	c := e.NewContext(req, httptest.NewRecorder())
	engine := &SsrEngineHTTPGateway{
		URL: "http://ssr.test",
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

	_, err := engine.Render(renderContext)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation error, got %v", err)
	}
}

func TestSsrEngineHTTPGateway_Render_InvalidPageJSON(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SsrResponse{
			Head: []string{"<title>Test</title>"},
			Body: "<div>Test</div>",
		})
	}))
	defer server.Close()

	engine := &SsrEngineHTTPGateway{
		URL:        server.URL,
		HttpClient: &http.Client{},
	}

	// Create render context with page that contains unmarshalable data
	ctx := &RenderContext{
		Page: &Page{
			Component: "TestComponent",
			Props:     map[string]any{"invalid": make(chan int)}, // channels can't be marshaled to JSON
			URL:       "/test",
			Version:   "1.0.0",
		},
	}

	_, err := engine.Render(ctx)
	if err == nil {
		t.Error("Expected JSON marshal error but got none")
	}
	if !strings.Contains(err.Error(), "failed to marshal page json") {
		t.Errorf("Expected error about JSON marshal, got: %v", err)
	}
}
