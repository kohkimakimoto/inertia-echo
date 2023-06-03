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

	"github.com/labstack/echo/v4"
)

// Renderer is a html/template renderer for Echo framework with inertia.js.
type Renderer struct {
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

func NewRenderer() *Renderer {
	r := &Renderer{
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

func (r *Renderer) AddViteEntryPoint(entryPoint ...string) {
	r.ViteEntryPoints = append(r.ViteEntryPoints, entryPoint...)
}

func (r *Renderer) Funcs(funcMap template.FuncMap) *Renderer {
	r.templates = r.templates.Funcs(funcMap)
	return r
}

func (r *Renderer) Parse(text string) (*Renderer, error) {
	t, err := r.templates.Parse(text)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *Renderer) MustParse(text string) *Renderer {
	t, err := r.Parse(text)
	if err != nil {
		panic(err)
	}
	return t
}

func (r *Renderer) ParseGlob(pattern string) (*Renderer, error) {
	t, err := r.templates.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *Renderer) MustParseGlob(pattern string) *Renderer {
	t, err := r.ParseGlob(pattern)
	if err != nil {
		panic(err)
	}
	return t
}

func (r *Renderer) ParseFS(f fs.FS, pattern string) (*Renderer, error) {
	t, err := r.templates.ParseFS(f, pattern)
	if err != nil {
		return nil, err
	}
	r.templates = t
	return r, nil
}

func (r *Renderer) MustParseFS(f fs.FS, pattern string) *Renderer {
	t, err := r.ParseFS(f, pattern)
	if err != nil {
		panic(err)
	}
	return t
}

// Render renders HTML by using templates.
func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	// The data is always a map[string]interface{}, if the renderer is used by Inertia.
	if mData, ok := data.(map[string]interface{}); ok {
		page, ok := mData["page"].(*Page)
		if !ok {
			return errors.New("page object is not found in the data")
		}

		in, err := Get(c)
		if err != nil {
			return err
		}

		if in.IsSsrEnabled() && r.SsrEngine != nil {
			// server-side rendering
			ssr, err := r.SsrEngine.Render(page)
			if err != nil {
				return err
			}
			mData["inertia"] = ssr.BodyHTML()
			mData["inertiaHead"] = ssr.HeadHTML()
		} else {
			// client-side rendering
			_inertia, err := r.renderInertia(page)
			if err != nil {
				return err
			}
			mData["inertia"] = _inertia
			mData["inertiaHead"] = ""
		}

		return r.templates.ExecuteTemplate(w, name, mData)

	}

	// The following is a fallback for the case that the renderer is used without Inertia.
	return r.templates.ExecuteTemplate(w, name, data)
}

func (r *Renderer) renderInertia(page *Page) (template.HTML, error) {
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

func (r *Renderer) funcMap() template.FuncMap {
	return template.FuncMap{
		// This function is a primitive way to render a data-page value for Inertia.
		// Generally, you don't have to use this function. You can use {{ .inertia }} instead.
		"json_marshal": r.fnJsonMarshal,
		// see https://vitejs.dev/guide/backend-integration.html
		"vite_react_refresh": r.fnReactRefresh,
		"vite":               r.fnVite,
	}
}

func (r *Renderer) fnJsonMarshal(v interface{}) (template.JS, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(j), nil
}

func (r *Renderer) fnReactRefresh() template.HTML {
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

func (r *Renderer) fnVite(entryPoints ...string) (template.HTML, error) {
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

func (r *Renderer) genTag(path string) string {
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

func (r *Renderer) ParseViteManifest(data []byte) error {
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

func (r *Renderer) MustParseViteManifest(data []byte) {
	if err := r.ParseViteManifest(data); err != nil {
		panic(err)
	}
}

func (r *Renderer) ParseViteManifestFile(name string) error {
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

func (r *Renderer) MustParseViteManifestFile(name string) {
	if err := r.ParseViteManifestFile(name); err != nil {
		panic(err)
	}
}

func (r *Renderer) ParseViteManifestFS(f fs.FS, name string) error {
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

func (r *Renderer) MustParseViteManifestFS(f fs.FS, name string) {
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
