package inertia

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

type Renderer interface {
	// Render renders a HTML for inertia.
	Render(io.Writer, string, map[string]interface{}, *Inertia) error
}

// HTMLRenderer is a html/template renderer for Echo framework with inertia.js.
type HTMLRenderer struct {
	templates   *template.Template
	Debug       bool
	ContainerId string

	// Vite integration

	Vite             bool
	ViteDevServerURL string
	ViteBasePath     string
	ViteDisableReact bool
	ViteEntryPoints  []string
	viteManifest     ViteManifest

	// SSR

	SsrEngine SsrEngine
}

func NewRenderer() *HTMLRenderer {
	r := &HTMLRenderer{
		Debug:            false,
		ContainerId:      "app",
		Vite:             true,
		ViteDevServerURL: "http://localhost:5173",
		ViteBasePath:     "/",
		ViteDisableReact: false,
		ViteEntryPoints:  []string{},
		viteManifest:     nil,
		SsrEngine:        nil,
	}
	r.templates = template.New("T").Funcs(r.funcMap())
	return r
}

func (r *HTMLRenderer) AddViteEntryPoint(entryPoint ...string) {
	r.ViteEntryPoints = append(r.ViteEntryPoints, entryPoint...)
}

func (r *HTMLRenderer) Funcs(funcMap template.FuncMap) *HTMLRenderer {
	r.templates = r.templates.Funcs(funcMap)
	return r
}

func (r *HTMLRenderer) Parse(text string) (*HTMLRenderer, error) {
	t, err := r.templates.Parse(text)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *HTMLRenderer) MustParse(text string) *HTMLRenderer {
	t, err := r.Parse(text)
	if err != nil {
		panic(err)
	}
	return t
}

func (r *HTMLRenderer) ParseGlob(pattern string) (*HTMLRenderer, error) {
	t, err := r.templates.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *HTMLRenderer) MustParseGlob(pattern string) *HTMLRenderer {
	t, err := r.ParseGlob(pattern)
	if err != nil {
		panic(err)
	}
	return t
}

func (r *HTMLRenderer) ParseFS(f fs.FS, pattern string) (*HTMLRenderer, error) {
	t, err := r.templates.ParseFS(f, pattern)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *HTMLRenderer) MustParseFS(f fs.FS, pattern string) *HTMLRenderer {
	t, err := r.ParseFS(f, pattern)
	if err != nil {
		panic(err)
	}
	return t
}

// Render renders HTML by using templates.
func (r *HTMLRenderer) Render(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
	page, ok := data["page"].(*Page)
	if !ok {
		return errors.New("page object is not found in the data")
	}

	if in.IsSsrEnabled() && r.SsrEngine != nil {
		// server-side rendering
		ssr, err := r.SsrEngine.Render(page)
		if err != nil {
			return err
		}
		data["inertia"] = ssr.BodyHTML()
		data["inertiaHead"] = ssr.HeadHTML()
	} else {
		// client-side rendering
		_inertia, err := r.renderInertia(page)
		if err != nil {
			return err
		}
		data["inertia"] = _inertia
		data["inertiaHead"] = ""
	}

	return r.templates.ExecuteTemplate(w, name, data)
}

func (r *HTMLRenderer) renderInertia(page *Page) (template.HTML, error) {
	pageJson, err := json.Marshal(page)
	if err != nil {
		return "", err
	}
	builder := new(strings.Builder)
	builder.WriteString(`<div id="` + r.ContainerId + `" data-page="`)
	template.HTMLEscape(builder, pageJson)
	builder.WriteString(`"></div>`)

	return template.HTML(builder.String()), nil
}

func (r *HTMLRenderer) funcMap() template.FuncMap {
	return template.FuncMap{
		// This function is a primitive way to render a data-page value for Inertia.
		// Generally, you don't have to use this function. You can use {{ .inertia }} instead.
		"json_marshal": r.fnJsonMarshal,
		// see https://vitejs.dev/guide/backend-integration.html
		"vite_react_refresh": r.fnReactRefresh,
		"vite":               r.fnVite,
	}
}

func (r *HTMLRenderer) fnJsonMarshal(v interface{}) (template.JS, error) {
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
		entryPoints = r.ViteEntryPoints
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

	if r.viteManifest == nil {
		return "", errors.New("manifest is not loaded")
	}

	tags := []string{}
	for _, entryPoint := range entryPoints {
		chunk, ok := r.viteManifest[entryPoint]
		if !ok {
			panic(fmt.Sprintf("unable to locate file in Vite manifest: %s", entryPoint))
		}

		if chunk, ok := chunk.(map[string]interface{}); ok {
			file := chunk["file"].(string)
			tags = append(tags, r.genTag(fmt.Sprintf("%s%s", r.ViteBasePath, file)))

			if css, ok := chunk["css"].([]interface{}); ok {
				for _, cssFile := range css {
					tags = append(tags, r.genTag(fmt.Sprintf("%s%s", r.ViteBasePath, cssFile)))
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

func (r *HTMLRenderer) ParseViteManifest(data []byte) error {
	if r.Debug {
		return nil
	}

	m, err := parseViteManifest(data)
	if err != nil {
		return err
	}
	r.viteManifest = m
	return nil
}

func (r *HTMLRenderer) MustParseViteManifest(data []byte) {
	if err := r.ParseViteManifest(data); err != nil {
		panic(err)
	}
}

func (r *HTMLRenderer) ParseViteManifestFile(name string) error {
	if r.Debug {
		return nil
	}

	m, err := parseViteManifestFile(name)
	if err != nil {
		return err
	}
	r.viteManifest = m
	return nil
}

func (r *HTMLRenderer) MustParseViteManifestFile(name string) {
	if err := r.ParseViteManifestFile(name); err != nil {
		panic(err)
	}
}

func (r *HTMLRenderer) ParseViteManifestFS(f fs.FS, name string) error {
	if r.Debug {
		return nil
	}

	m, err := parseViteManifestFS(f, name)
	if err != nil {
		return err
	}
	r.viteManifest = m
	return nil
}

func (r *HTMLRenderer) MustParseViteManifestFS(f fs.FS, name string) {
	if err := r.ParseViteManifestFS(f, name); err != nil {
		panic(err)
	}
}

type ViteManifest map[string]interface{}

func parseViteManifest(data []byte) (ViteManifest, error) {
	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func parseViteManifestFile(name string) (ViteManifest, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return parseViteManifest(b)
}

func parseViteManifestFS(f fs.FS, name string) (ViteManifest, error) {
	b, err := fs.ReadFile(f, name)
	if err != nil {
		return nil, err
	}
	return parseViteManifest(b)
}
