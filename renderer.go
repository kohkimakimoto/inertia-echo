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
//   https://inertiajs.com/the-protocol#the-page-object
//   https://echo.labstack.com/guide/templates/
//
// Notice:
// It is a built-in renderer included in the inertia-echo.
// But you don't have to use it. You can use any renderers you want with inertia-echo.
// The inertia-echo is renderer agnostic.
type Renderer struct {
	templates *template.Template
}

// NewRenderer returns a new Renderer instance.
func NewRenderer(pattern string, funcMap template.FuncMap) *Renderer {
	return &Renderer{
		templates: template.Must(template.New("template").Funcs(mergeFuncMap(builtinFuncMap, funcMap)).ParseGlob(pattern)),
	}
}

// NewRendererWithFS returns a new Renderer instance with FS interface.
func NewRendererWithFS(f fs.FS, pattern string, funcMap template.FuncMap) *Renderer {
	return &Renderer{
		templates: template.Must(template.New("template").Funcs(mergeFuncMap(builtinFuncMap, funcMap)).ParseFS(f, pattern)),
	}
}

// Render renders HTML by using templates.
func (t *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var builtinFuncMap = template.FuncMap{
	"json_marshal": jsonMarshal,
}

func jsonMarshal(v interface{}) template.JS {
	ret, _ := json.Marshal(v)
	return template.JS(ret)
}

func mergeFuncMap(funcMaps ...template.FuncMap) template.FuncMap {
	merged := template.FuncMap{}
	for _, funcMap := range funcMaps {
		if funcMap != nil {
			for k, v := range funcMap {
				merged[k] = v
			}
		}
	}
	return merged
}
