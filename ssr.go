package inertia

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// SsrGateway dispatches a Page to an SSR backend.
// Named SsrGateway because this package has no Ssr subpackage (cf. Inertia\Ssr\Gateway).
//
// Contract:
//   - (resp, nil)  — SSR HTML
//   - (nil, nil)   — intentional skip (caller uses CSR); not an error
//   - (nil, err)   — failure; caller handles (no built-in CSR fallback)
type SsrGateway interface {
	Dispatch(ctx *RenderContext) (*SsrResponse, error)
}

// SsrResponse is the HTML payload returned by an SSR backend.
// Named with an Ssr prefix to avoid colliding with net/http or Echo response types.
// Mirrors inertia-laravel's Inertia\Ssr\Response.
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

// SsrHTTPGateway is a shared HTTP client for an SSR server (usually Node).
// It is the transport + process-level switches only — not a request policy object.
//
// Compared with inertia-laravel HttpGateway:
//   - Request skip policy (withoutSsr / disableSsr) lives on Inertia; see SsrEnabled().
//     Laravel mutates a container Gateway; this type must stay safe to share across requests.
//   - Failures return error (Go). Laravel defaults to null CSR + optional SsrRenderFailed /
//     throw_on_error. Wrap SsrGateway or use a custom Renderer for CSR-on-failure.
//   - No SsrState: HTMLRenderer calls Dispatch once for head and body.
type SsrHTTPGateway struct {
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

// NewSsrHTTPGateway returns an SsrHTTPGateway with defaults aligned to Laravel's SSR URL.
func NewSsrHTTPGateway() *SsrHTTPGateway {
	return &SsrHTTPGateway{
		Enabled:    true,
		URL:        "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

// Dispatch POSTs the page JSON to the SSR server.
func (g *SsrHTTPGateway) Dispatch(ctx *RenderContext) (*SsrResponse, error) {
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
func (g *SsrHTTPGateway) IsHealthy(ctx context.Context) bool {
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

func (g *SsrHTTPGateway) bundleExists() bool {
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

func (g *SsrHTTPGateway) post(ctx *RenderContext) (*SsrResponse, error) {
	if ctx == nil || ctx.Page == nil {
		return nil, fmt.Errorf("ssr: render context or page is nil")
	}

	pageJSON, err := json.Marshal(ctx.Page)
	if err != nil {
		return nil, fmt.Errorf("ssr: failed to marshal page json: %w", err)
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
	// JSON null — intentional skip (e.g. Vite SSR warming up), same idea as Laravel.
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

func (g *SsrHTTPGateway) client() *http.Client {
	if g.HttpClient != nil {
		return g.HttpClient
	}
	return http.DefaultClient
}

func requestContextFrom(ctx *RenderContext) context.Context {
	if ctx != nil && ctx.Inertia != nil {
		if ec := ctx.Inertia.EchoContext(); ec != nil && ec.Request() != nil {
			return ec.Request().Context()
		}
	}
	return context.Background()
}

func (g *SsrHTTPGateway) endpoint() string {
	if g.Endpoint != "" {
		return g.Endpoint
	}
	if g.UseHot {
		return g.hotURL("/__inertia_ssr")
	}
	return g.productionURL("/render")
}

func (g *SsrHTTPGateway) productionURL(pathSuffix string) string {
	return strings.TrimRight(g.URL, "/") + ensureLeadingSlash(pathSuffix)
}

func (g *SsrHTTPGateway) hotURL(pathSuffix string) string {
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

// matchSSRPaths reports whether path matches any exclusion pattern.
// Used by Inertia.SsrEnabled for WithoutSsr (Laravel Request::is / ExcludesPaths style).
func matchSSRPaths(path string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, pattern := range patterns {
		if matchSSRPath(path, pattern) {
			return true
		}
	}
	return false
}

func matchSSRPath(path, pattern string) bool {
	if pattern == "" {
		return false
	}
	trimmedPath := strings.Trim(path, "/")
	trimmedPattern := pattern
	if pattern != "/" {
		trimmedPattern = strings.Trim(pattern, "/")
	} else {
		return trimmedPath == ""
	}
	return matchStarPattern(trimmedPattern, trimmedPath) || matchStarPattern(pattern, path)
}

func matchStarPattern(pattern, value string) bool {
	if pattern == value {
		return true
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); {
		if pattern[i] == '*' {
			b.WriteString(".*")
			i++
			continue
		}
		j := i
		for j < len(pattern) && pattern[j] != '*' {
			j++
		}
		b.WriteString(regexp.QuoteMeta(pattern[i:j]))
		i = j
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return false
	}
	return re.MatchString(value)
}
