package inertia

import (
	"embed"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

//go:embed testdata
var testdataFs embed.FS

func TestRenderer_Render(t *testing.T) {
	expected := `<div id="app" data-page="{&#34;component&#34;:&#34;Page&#34;,&#34;props&#34;:{&#34;title&#34;:&#34;Hello, World!&#34;}}"></div>`

	t.Run("NewRenderer", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		e.Renderer = NewRenderer(filepath.Join("testdata", "*.html"), nil)
		c := e.NewContext(req, rec)
		err := c.Render(http.StatusOK, "app.html", map[string]interface{}{
			"page": map[string]interface{}{
				"component": "Page",
				"props":     map[string]interface{}{"title": "Hello, World!"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("NewRendererWithFS", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		e.Renderer = NewRendererWithFS(testdataFs, filepath.Join("testdata", "*.html"), nil)
		c := e.NewContext(req, rec)
		err := c.Render(http.StatusOK, "app.html", map[string]interface{}{
			"page": map[string]interface{}{
				"component": "Page",
				"props":     map[string]interface{}{"title": "Hello, World!"},
			},
		})
		if err != nil {
			t.Fatal(err)
		}
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s", body)
		}
	})
}
