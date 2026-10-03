package inertia

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/labstack/echo/v5"
)

// SsrGateway dispatches a Page to an SSR backend.
// Mirrors inertia-laravel's Inertia\Ssr\Gateway (named SsrGateway because this
// package has no Ssr subpackage / namespace).
// A nil response with a nil error means SSR was skipped (use CSR).
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

// SsrHTTPGateway is the HTTP implementation of SsrGateway.
// Mirrors inertia-laravel's Inertia\Ssr\HttpGateway.
//
// Intentional differences from inertia-laravel:
//   - Base settings live on this shared struct; per-request withoutSsr / disableSsr
//     are stored on the request-scoped Inertia and read in Dispatch (Go concurrency).
//   - Failed SSR returns an error by default; set FallbackOnError for Laravel-like CSR fallback.
//   - No SsrState: HTMLRenderer calls Dispatch once for both head and body.
type SsrHTTPGateway struct {
	// Enabled controls whether SSR is on. Default true (config inertia.ssr.enabled).
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
	// Except is the startup-time path exclusion list (immutable during requests).
	// Per-request exclusions go on Inertia via WithoutSsr (see MiddlewareConfig.WithoutSsr).
	Except []string
	// Disable disables SSR when the function returns true (base condition).
	Disable func(c *echo.Context) bool
	// EnsureBundleExists skips SSR when none of BundlePaths exist on disk.
	EnsureBundleExists bool
	// BundlePaths are candidate SSR bundle file paths checked when EnsureBundleExists is true.
	BundlePaths []string
	// FallbackOnError falls back to CSR (nil, nil) on SSR failure. Default false.
	// Unlike inertia-laravel (graceful CSR by default), errors propagate unless this is set.
	FallbackOnError bool
	// ErrorReporter is called when an SSR request fails, before fallback or return.
	ErrorReporter func(ctx *RenderContext, err error)
	// HttpClient communicates with the SSR server. Timeout is disabled by default.
	HttpClient *http.Client

	exceptMu sync.Mutex
}

// NewSsrHTTPGateway returns an SsrHTTPGateway with Laravel-compatible defaults.
func NewSsrHTTPGateway() *SsrHTTPGateway {
	return &SsrHTTPGateway{
		Enabled:    true,
		URL:        "http://127.0.0.1:13714",
		HttpClient: &http.Client{},
	}
}

// ExceptPaths appends paths to the base Except list (startup configuration).
// Mirrors inertia-laravel ExcludesSsrPaths::except for shared gateway config.
// Do not call this per request; use WithoutSsr on Inertia instead.
func (g *SsrHTTPGateway) ExceptPaths(paths ...string) {
	g.exceptMu.Lock()
	defer g.exceptMu.Unlock()
	g.Except = append(g.Except, paths...)
}

// Dispatch sends the page to the SSR server, or returns (nil, nil) when SSR is skipped.
func (g *SsrHTTPGateway) Dispatch(ctx *RenderContext) (*SsrResponse, error) {
	if !g.ssrIsEnabled(ctx) {
		return nil, nil
	}
	if g.EnsureBundleExists && !g.bundleExists() {
		return nil, nil
	}

	resp, err := g.post(ctx)
	if err != nil {
		return g.handleFailure(ctx, err)
	}
	return resp, nil
}

// IsHealthy reports whether the production SSR server responds to GET /health.
// Mirrors inertia-laravel HasHealthCheck::isHealthy.
func (g *SsrHTTPGateway) IsHealthy(ctx context.Context) bool {
	client := g.HttpClient
	if client == nil {
		client = http.DefaultClient
	}
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

func (g *SsrHTTPGateway) ssrIsEnabled(ctx *RenderContext) bool {
	if !g.Enabled {
		return false
	}

	var c *echo.Context
	var path string
	if ctx != nil && ctx.Inertia != nil {
		c = ctx.Inertia.EchoContext()
		if ctx.Inertia.ssrDisabled {
			return false
		}
		if ctx.Inertia.ssrDisableWhen != nil && c != nil && ctx.Inertia.ssrDisableWhen(c) {
			return false
		}
		if c != nil && c.Request() != nil {
			path = c.Request().URL.Path
		}
		if matchSSRPaths(path, ctx.Inertia.withoutSsr) {
			return false
		}
	}

	g.exceptMu.Lock()
	except := append([]string(nil), g.Except...)
	g.exceptMu.Unlock()
	if matchSSRPaths(path, except) {
		return false
	}
	if g.Disable != nil && c != nil && g.Disable(c) {
		return false
	}
	return true
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
		g.endpoint(),
		bytes.NewReader(pageJSON),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := g.HttpClient
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

func (g *SsrHTTPGateway) handleFailure(ctx *RenderContext, err error) (*SsrResponse, error) {
	if g.ErrorReporter != nil {
		g.ErrorReporter(ctx, err)
	}
	if ctx != nil && ctx.Inertia != nil {
		echoContext := ctx.Inertia.EchoContext()
		if echoContext != nil && echoContext.Request() != nil && echoContext.Request().Context().Err() != nil {
			return nil, err
		}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return nil, err
	}
	if g.FallbackOnError {
		return nil, nil
	}
	return nil, err
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
// Patterns follow Laravel Request::is / ExcludesPaths style (* wildcards).
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
