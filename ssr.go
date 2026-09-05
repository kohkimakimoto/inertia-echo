package inertia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
)

type SsrEngine interface {
	Render(ctx *RenderContext) (*SsrResponse, error)
}

type SsrResponse struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

func (r *SsrResponse) HeadHTML() template.HTML {
	return template.HTML(strings.Join(r.Head, "\n"))
}

func (r *SsrResponse) BodyHTML() template.HTML {
	return template.HTML(r.Body)
}

// SsrEngineHTTPGateway is an SSR engine that communicates with a remote SSR server over HTTP.
// The server is usually a Node.js server.
type SsrEngineHTTPGateway struct {
	// Server URL. When Endpoint is empty, requests are sent to URL + "/render".
	URL string
	// Endpoint is the complete SSR endpoint URL. When set, it takes precedence
	// over URL. This supports endpoints such as Vite's /__inertia_ssr.
	Endpoint string
	// HTTP client to communicate with the SSR server.
	// Its timeout is disabled by default and can be configured by the application.
	HttpClient *http.Client
}

func NewSsrEngineHTTPGateway() *SsrEngineHTTPGateway {
	return &SsrEngineHTTPGateway{
		URL:        "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

func (s *SsrEngineHTTPGateway) Render(ctx *RenderContext) (*SsrResponse, error) {
	pageJSON, err := json.Marshal(ctx.Page)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal page json: %w", err)
	}

	requestContext := context.Background()
	if ctx.Inertia != nil {
		echoContext := ctx.Inertia.EchoContext()
		if echoContext != nil && echoContext.Request() != nil {
			requestContext = echoContext.Request().Context()
		}
	}

	req, err := http.NewRequestWithContext(
		requestContext,
		http.MethodPost,
		s.endpoint(),
		bytes.NewReader(pageJSON),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := s.HttpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssr: status code is not 200: %d", resp.StatusCode)
	}

	var payload *struct {
		Head []string `json:"head"`
		Body *string  `json:"body"`
	}
	responseJSON, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ssr: failed to read response: %w", err)
	}
	if err := json.Unmarshal(responseJSON, &payload); err != nil {
		return nil, fmt.Errorf("ssr: failed to decode response: %w", err)
	}
	if payload == nil {
		return nil, nil
	}
	if payload.Body == nil {
		return nil, fmt.Errorf("ssr: response body is missing or null")
	}

	return &SsrResponse{
		Head: payload.Head,
		Body: *payload.Body,
	}, nil
}

func (s *SsrEngineHTTPGateway) endpoint() string {
	if s.Endpoint != "" {
		return s.Endpoint
	}
	return strings.TrimRight(s.URL, "/") + "/render"
}
