package inertia

import (
	"html/template"
	"regexp"
	"strings"
)

// SsrGateway dispatches a Page to an SSR backend.
//
// Built-in implementations:
//   - [github.com/kohkimakimoto/inertia-echo/v5/ssr/nodejs] — HTTP client for a Node SSR server
//   - [github.com/kohkimakimoto/inertia-echo/v5/ssr/slowjs] — in-process slowjs VM
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
