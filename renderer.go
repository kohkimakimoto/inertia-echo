package inertia

import (
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/labstack/echo/v4"
)

// Renderer is a html/template renderer for Echo framework with inertia.js.
// It is a built-in renderer included in the inertia-echo.
// But you don't have to use it. You can use any renderers you want with inertia-echo.
type Renderer struct {
	templates   *template.Template
	containerId string
}

type RendererConfig struct {
	ContainerId string
}

var DefaultRendererConfig = RendererConfig{
	ContainerId: "app",
}

func NewRendererWithConfig(config RendererConfig) *Renderer {
	if config.ContainerId == "" {
		config.ContainerId = DefaultRendererConfig.ContainerId
	}

	return &Renderer{
		templates:   template.New("T").Funcs(builtinFuncMap),
		containerId: config.ContainerId,
	}
}

func NewRenderer() *Renderer {
	return NewRendererWithConfig(DefaultRendererConfig)
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
	if m, ok := data.(map[string]interface{}); ok {
		// The data is always a map[string]interface{}, if the renderer is used by Inertia.
		page, ok := m["page"].(*Page)
		if !ok {
			return errors.New("page object is not found in the data")
		}
		_inertia, err := r.renderInertia(page)
		if err != nil {
			return err
		}
		m["inertia"] = _inertia
		return r.templates.ExecuteTemplate(w, name, m)
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
	builder.WriteString(`<div id="` + r.containerId + `" data-page="`)
	template.HTMLEscape(builder, pageJson)
	builder.WriteString(`"></div>`)

	return template.HTML(builder.String()), nil
}

var builtinFuncMap = template.FuncMap{
	// This function is a primitive way to render a data-page value for Inertia.
	// Generally, you don't have to use this function. You can use {{ .inertia }} instead.
	"json_marshal": fnJsonMarshal,
}

func fnJsonMarshal(v interface{}) (template.JS, error) {
	j, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return template.JS(j), nil
}
