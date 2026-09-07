package inertia

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"path"
	"regexp"
	"strings"
)

type Renderer interface {
	// Render renders an HTML page for inertia.
	Render(ctx *RenderContext) error
}

// HTMLRenderer is a html/template renderer for Echo framework with inertia.js.
type HTMLRenderer struct {
	// Templates is the parsed html/template set used by Render.
	// Prefer ParseGlob, ParseFS, or ParseFiles (or their Must* variants), which apply FuncMap and set this field.
	// You may also build a template set yourself with FuncMap and assign it here.
	Templates *template.Template
	// Debug enables Vite development mode helpers (dev server tags and React refresh).
	// When true, vite template functions talk to ViteDevServerURL instead of ViteManifest.
	Debug bool
	// ContainerId is the DOM id of the Inertia root element (and data-page script target).
	// Defaults to "app".
	ContainerId string

	// ViteDevServerURL is the origin of the Vite development server.
	// Used when Debug is true. Defaults to "http://localhost:5173".
	ViteDevServerURL string
	// ViteBasePath is the URL path prefix for built assets from ViteManifest.
	// Used when Debug is false. Defaults to "/".
	ViteBasePath string
	// ViteManifest is the production Vite manifest. Required when Debug is false.
	// Build it with ParseViteManifest* helpers (or your own loader) and assign it here.
	ViteManifest ViteManifest

	// SsrEngine renders the page on the server when SSR is enabled for the request.
	// If nil, SSR is skipped and the CSR bootstrap is used.
	SsrEngine SsrEngine
	// SsrFallbackOnError falls back to the CSR bootstrap when SsrEngine returns an error.
	// Request cancellation is never hidden by this fallback.
	SsrFallbackOnError bool
	// SsrErrorReporter is called when SsrEngine returns an error, before fallback or return.
	SsrErrorReporter func(ctx *RenderContext, err error)
}

func NewHTMLRenderer() *HTMLRenderer {
	return &HTMLRenderer{
		Debug:              false,
		ContainerId:        "app",
		ViteDevServerURL:   "http://localhost:5173",
		ViteBasePath:       "/",
		ViteManifest:       nil,
		SsrEngine:          nil,
		SsrFallbackOnError: false,
		SsrErrorReporter:   nil,
	}
}

// FuncMap returns template functions required by Inertia/Vite helpers
// (json_marshal, vite, vite_react_refresh).
// Prefer ParseGlob, ParseFS, or ParseFiles, which apply this map automatically.
// Use FuncMap directly when you build templates yourself with html/template.
func (r *HTMLRenderer) FuncMap() template.FuncMap {
	return template.FuncMap{
		// This function is a primitive way to render a data-page value for Inertia.
		// Generally, you don't have to use this function. You can use {{ .inertia }} instead.
		"json_marshal": r.fnJsonMarshal,
		// see https://vitejs.dev/guide/backend-integration.html
		"vite_react_refresh": r.fnReactRefresh,
		"vite":               r.fnVite,
	}
}

// ParseFiles parses the named files with FuncMap applied and sets Templates.
// It mirrors html/template.ParseFiles.
func (r *HTMLRenderer) ParseFiles(filenames ...string) (*template.Template, error) {
	t, err := template.New("").Funcs(r.FuncMap()).ParseFiles(filenames...)
	if err != nil {
		return nil, err
	}
	r.Templates = t
	return t, nil
}

// MustParseFiles is like ParseFiles but panics on error.
func (r *HTMLRenderer) MustParseFiles(filenames ...string) *template.Template {
	t, err := r.ParseFiles(filenames...)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseGlob parses files matching the pattern with FuncMap applied and sets Templates.
// It mirrors html/template.ParseGlob.
func (r *HTMLRenderer) ParseGlob(pattern string) (*template.Template, error) {
	t, err := template.New("").Funcs(r.FuncMap()).ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	r.Templates = t
	return t, nil
}

// MustParseGlob is like ParseGlob but panics on error.
func (r *HTMLRenderer) MustParseGlob(pattern string) *template.Template {
	t, err := r.ParseGlob(pattern)
	if err != nil {
		panic(err)
	}
	return t
}

// ParseFS parses files from fsys matching the patterns with FuncMap applied and sets Templates.
// It mirrors html/template.ParseFS.
func (r *HTMLRenderer) ParseFS(fsys fs.FS, patterns ...string) (*template.Template, error) {
	t, err := template.New("").Funcs(r.FuncMap()).ParseFS(fsys, patterns...)
	if err != nil {
		return nil, err
	}
	r.Templates = t
	return t, nil
}

// MustParseFS is like ParseFS but panics on error.
func (r *HTMLRenderer) MustParseFS(fsys fs.FS, patterns ...string) *template.Template {
	t, err := r.ParseFS(fsys, patterns...)
	if err != nil {
		panic(err)
	}
	return t
}

// Render renders HTML by using templates.
func (r *HTMLRenderer) Render(ctx *RenderContext) error {
	if r.Templates == nil {
		return errors.New("HTMLRenderer: Templates is nil")
	}

	var data map[string]any
	if ctx.ViewData != nil {
		_data, ok := ctx.ViewData.(map[string]any)
		if !ok {
			return errors.New("HTMLRenderer requires ViewData to be a map[string]any")
		}
		data = make(map[string]any, len(_data)+3)
		for key, value := range _data {
			data[key] = value
		}
	} else {
		data = map[string]any{}
	}

	data["page"] = ctx.Page

	if ctx.Inertia.IsSsrEnabled() && r.SsrEngine != nil {
		ssr, err := r.SsrEngine.Render(ctx)
		if err != nil {
			if r.SsrErrorReporter != nil {
				r.SsrErrorReporter(ctx, err)
			}
			if !r.SsrFallbackOnError || ctx.Inertia.EchoContext().Request().Context().Err() != nil {
				return err
			}
		} else if ssr != nil {
			data["inertia"] = ssr.BodyHTML()
			data["inertiaHead"] = ssr.HeadHTML()
			return r.Templates.ExecuteTemplate(ctx.Writer, ctx.ViewName, data)
		}
	}

	_inertia, err := r.renderInertia(ctx.Page)
	if err != nil {
		return err
	}
	data["inertia"] = _inertia
	data["inertiaHead"] = ""

	return r.Templates.ExecuteTemplate(ctx.Writer, ctx.ViewName, data)
}

func (r *HTMLRenderer) renderInertia(page *Page) (template.HTML, error) {
	pageJson, err := json.Marshal(page)
	if err != nil {
		return "", err
	}
	builder := new(strings.Builder)
	id := template.HTMLEscapeString(r.ContainerId)
	builder.WriteString(`<script type="application/json" data-page="`)
	builder.WriteString(id)
	builder.WriteString(`">`)
	builder.Write(pageJson)
	builder.WriteString(`</script><div id="`)
	builder.WriteString(id)
	builder.WriteString(`"></div>`)

	return template.HTML(builder.String()), nil
}

func (r *HTMLRenderer) fnJsonMarshal(v any) (template.JS, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(j), nil
}

func (r *HTMLRenderer) fnReactRefresh() template.HTML {
	if !r.Debug {
		return ""
	}

	return template.HTML(fmt.Sprintf(`<script type="module">
  import RefreshRuntime from '%s/@react-refresh'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>`, r.ViteDevServerURL))
}

func (r *HTMLRenderer) fnVite(entryPoints ...string) (template.HTML, error) {
	if len(entryPoints) == 0 {
		return "", errors.New("vite: at least one entry point is required")
	}

	if r.Debug {
		tags := []string{
			fmt.Sprintf(`<script type="module" src="%s/@vite/client"></script>`, r.ViteDevServerURL),
		}
		for _, entryPoint := range entryPoints {
			tags = append(tags, r.genTag(fmt.Sprintf("%s/%s", r.ViteDevServerURL, entryPoint)))
		}
		return template.HTML(strings.Join(tags, "")), nil
	}

	if r.ViteManifest == nil {
		return "", errors.New("manifest is not loaded")
	}

	tags := []string{}
	for _, entryPoint := range entryPoints {
		chunk, ok := r.ViteManifest[entryPoint]
		if !ok {
			panic(fmt.Sprintf("unable to locate file in Vite manifest: %s", entryPoint))
		}

		if chunk, ok := chunk.(map[string]any); ok {
			file := chunk["file"].(string)
			tags = append(tags, r.genTag(path.Join(r.ViteBasePath, file)))

			if cssList, ok := chunk["css"].([]any); ok {
				for _, cssV := range cssList {
					cssFile, ok := cssV.(string)
					if !ok {
						return "", fmt.Errorf("the Vite manifest has an invalid css file: %v", cssV)
					}
					tags = append(tags, r.genTag(path.Join(r.ViteBasePath, cssFile)))
				}
			}
		}
	}
	return template.HTML(strings.Join(tags, "")), nil
}

func (r *HTMLRenderer) genTag(path string) string {
	if isCssPath(path) {
		return fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, path)
	} else {
		return fmt.Sprintf(`<script type="module" src="%s"></script>`, path)
	}
}

var cssRe = regexp.MustCompile(`\.(css|less|sass|scss|styl|stylus|pcss|postcss)$`)

func isCssPath(name string) bool {
	return cssRe.MatchString(name)
}
