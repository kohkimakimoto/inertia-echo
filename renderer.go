package inertia

import (
	"encoding/json"
	"html/template"
	"io"
	"io/fs"

	"github.com/labstack/echo/v4"
)

// Renderer is a html/template renderer for Echo framework.
// It provides `json_marshal` template function to render a JSON encoded page object.
// see also:
//
//	https://inertiajs.com/the-protocol#the-page-object
//	https://echo.labstack.com/guide/templates/
//
// Notice:
// It is a built-in renderer included in the inertia-echo.
// But you don't have to use it. You can use any renderers you want with inertia-echo.
// The inertia-echo is renderer agnostic.
type Renderer struct {
	templates *template.Template
}

func NewRenderer() *Renderer {
	return &Renderer{
		templates: template.New("T").Funcs(builtinFuncMap),
	}
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
	return r.templates.ExecuteTemplate(w, name, data)
}

var builtinFuncMap = template.FuncMap{
	"json_marshal": fnJsonMarshal,
}

func fnJsonMarshal(v interface{}) template.JS {
	ret, _ := json.Marshal(v)
	return template.JS(ret)
}
