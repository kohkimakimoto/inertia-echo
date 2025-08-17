package inertia

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSsrEngineHTTPGateway_Render(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   SsrResponse
		mockStatusCode int
		expectError    bool
		errorContains  string
	}{
		{
			name: "successful render",
			mockResponse: SsrResponse{
				Head: []string{"<title>Test Page</title>", "<meta name=\"description\" content=\"test\">"},
				Body: "<div>Test Content</div>",
			},
			mockStatusCode: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "server error",
			mockStatusCode: http.StatusInternalServerError,
			expectError:    true,
			errorContains:  "status code is not 200: 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request body contains valid page JSON
				var page Page
				if err := json.NewDecoder(r.Body).Decode(&page); err != nil {
					t.Errorf("Failed to decode request body: %v", err)
				}

				// Set response status
				w.WriteHeader(tt.mockStatusCode)

				// Return response based on test case
				if tt.mockStatusCode == http.StatusOK {
					json.NewEncoder(w).Encode(tt.mockResponse)
				}
			}))
			defer server.Close()

			// Create SSR engine with mock server URL
			engine := &SsrEngineHTTPGateway{
				URL:        server.URL,
				HttpClient: &http.Client{},
			}

			// Create test render context
			ctx := &RenderContext{
				Page: &Page{
					Component: "TestComponent",
					Props:     map[string]any{"key": "value"},
					URL:       "/test",
					Version:   "1.0.0",
				},
			}

			// Execute render
			result, err := engine.Render(ctx)

			// Check error expectations
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			// Check success case
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			// Verify response content
			if len(result.Head) != len(tt.mockResponse.Head) {
				t.Errorf("Expected %d head elements, got %d", len(tt.mockResponse.Head), len(result.Head))
			}
			for i, head := range tt.mockResponse.Head {
				if i < len(result.Head) && result.Head[i] != head {
					t.Errorf("Expected head[%d] to be '%s', got '%s'", i, head, result.Head[i])
				}
			}

			if result.Body != tt.mockResponse.Body {
				t.Errorf("Expected body '%s', got '%s'", tt.mockResponse.Body, result.Body)
			}
		})
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

	if engine.HttpClient == nil {
		t.Error("Expected non-nil HTTP client")
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
