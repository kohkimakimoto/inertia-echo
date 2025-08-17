package inertia

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
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
	// Server URL
	URL string
	// HTTP client to communicate with the SSR server
	HttpClient *http.Client
}

func NewSsrEngineHTTPGateway() *SsrEngineHTTPGateway {
	return &SsrEngineHTTPGateway{
		URL:        "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

func (s *SsrEngineHTTPGateway) Render(ctx *RenderContext) (*SsrResponse, error) {
	pJson, err := json.Marshal(ctx.Page)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal page json: %w", err)
	}

	req, err := http.NewRequest(
		http.MethodPost,
		s.URL+"/render",
		bytes.NewBuffer(pJson),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.HttpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssr: status code is not 200: %d", resp.StatusCode)
	}

	var ssrResp SsrResponse
	err = json.NewDecoder(resp.Body).Decode(&ssrResp)
	if err != nil {
		return nil, err
	}
	return &ssrResp, nil
}
