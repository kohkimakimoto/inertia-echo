package nodejs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
)

// Gateway is a shared HTTP client for an SSR server (usually Node).
// It is the transport + process-level switches only — not a request policy object.
//
// Compared with inertia-laravel HttpGateway:
//   - Request skip policy (withoutSsr / disableSsr) lives on Inertia; see SsrEnabled().
//     Laravel mutates a container Gateway; this type must stay safe to share across requests.
//   - Failures return error (Go). Laravel defaults to null CSR + optional SsrRenderFailed /
//     throw_on_error. Wrap SsrGateway or use a custom Renderer for CSR-on-failure.
//   - No SsrState: HTMLRenderer calls Dispatch once for head and body.
type Gateway struct {
	// Enabled is the process-level SSR switch (config inertia.ssr.enabled). Default true.
	// Request-level off switches use Inertia.DisableSsr / WithoutSsr, not this field.
	Enabled bool
	// URL is the production SSR server base. When Endpoint is empty and UseHot is false,
	// requests go to URL + "/render".
	URL string
	// HotURL is the Vite hot origin. Used when UseHot is true (HotURL + "/__inertia_ssr").
	HotURL string
	// UseHot selects the Vite hot SSR endpoint instead of the production URL.
	UseHot bool
	// Endpoint is the complete SSR endpoint URL. When set, it takes precedence over URL/HotURL.
	Endpoint string
	// EnsureBundleExists skips Dispatch with (nil, nil) when no BundlePaths entry exists.
	// Mirrors config inertia.ssr.ensure_bundle_exists (soft skip, not an error).
	EnsureBundleExists bool
	// BundlePaths are candidate SSR bundle file paths for EnsureBundleExists.
	BundlePaths []string
	// HttpClient communicates with the SSR server. Timeout is disabled by default.
	HttpClient *http.Client
}

// NewGateway returns a Gateway with defaults aligned to Laravel's SSR URL.
func NewGateway() *Gateway {
	return &Gateway{
		Enabled:    true,
		URL:        "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

// Dispatch POSTs the page JSON to the SSR server.
func (g *Gateway) Dispatch(ctx *inertia.RenderContext) (*inertia.SsrResponse, error) {
	if !g.Enabled {
		return nil, nil
	}
	// Honor request policy when called with an Inertia context (also checked in HTMLRenderer).
	if ctx != nil && ctx.Inertia != nil && !ctx.Inertia.SsrEnabled() {
		return nil, nil
	}
	if g.EnsureBundleExists && !g.bundleExists() {
		return nil, nil
	}
	return g.post(ctx)
}

// IsHealthy reports whether the production SSR server responds to GET /health.
// Mirrors inertia-laravel HasHealthCheck::isHealthy.
func (g *Gateway) IsHealthy(ctx context.Context) bool {
	client := g.client()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, g.productionURL("/health"), nil)
	if err != nil {
		return false
	}
	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func (g *Gateway) bundleExists() bool {
	for _, p := range g.BundlePaths {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func (g *Gateway) post(ctx *inertia.RenderContext) (*inertia.SsrResponse, error) {
	if ctx == nil || ctx.Page == nil {
		return nil, fmt.Errorf("ssr/nodejs: render context or page is nil")
	}

	pageJSON, err := json.Marshal(ctx.Page)
	if err != nil {
		return nil, fmt.Errorf("ssr/nodejs: failed to marshal page json: %w", err)
	}

	req, err := http.NewRequestWithContext(
		requestContextFrom(ctx),
		http.MethodPost,
		g.endpoint(),
		bytes.NewReader(pageJSON),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ssr/nodejs: status code is not 200: %d", resp.StatusCode)
	}

	var payload *struct {
		Head []string `json:"head"`
		Body *string  `json:"body"`
	}
	responseJSON, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ssr/nodejs: failed to read response: %w", err)
	}
	if err := json.Unmarshal(responseJSON, &payload); err != nil {
		return nil, fmt.Errorf("ssr/nodejs: failed to decode response: %w", err)
	}
	// JSON null — intentional skip (e.g. Vite SSR warming up), same idea as Laravel.
	if payload == nil {
		return nil, nil
	}
	if payload.Body == nil {
		return nil, fmt.Errorf("ssr/nodejs: response body is missing or null")
	}

	return &inertia.SsrResponse{
		Head: payload.Head,
		Body: *payload.Body,
	}, nil
}

func (g *Gateway) client() *http.Client {
	if g.HttpClient != nil {
		return g.HttpClient
	}
	return http.DefaultClient
}

func requestContextFrom(ctx *inertia.RenderContext) context.Context {
	if ctx != nil && ctx.Inertia != nil {
		if ec := ctx.Inertia.EchoContext(); ec != nil && ec.Request() != nil {
			return ec.Request().Context()
		}
	}
	return context.Background()
}

func (g *Gateway) endpoint() string {
	if g.Endpoint != "" {
		return g.Endpoint
	}
	if g.UseHot {
		return g.hotURL("/__inertia_ssr")
	}
	return g.productionURL("/render")
}

func (g *Gateway) productionURL(pathSuffix string) string {
	return strings.TrimRight(g.URL, "/") + ensureLeadingSlash(pathSuffix)
}

func (g *Gateway) hotURL(pathSuffix string) string {
	base := g.HotURL
	if base == "" {
		base = g.URL
	}
	return strings.TrimRight(base, "/") + ensureLeadingSlash(pathSuffix)
}

func ensureLeadingSlash(p string) string {
	if p == "" {
		return "/"
	}
	if strings.HasPrefix(p, "/") {
		return p
	}
	return "/" + p
}

// Compile-time check.
var _ inertia.SsrGateway = (*Gateway)(nil)
