package inertia

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestInertia_Version(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	in := New(c, "app.html", map[string]interface{}{}, nil)

	in.SetVersion(func() string {
		return "123456789"
	})

	v := in.Version()
	if v != "123456789" {
		t.Errorf("inertia.Version() = %v, want %v", v, "1")
	}
}

func TestInertia_SetRootView(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	in := New(c, "app.html", map[string]interface{}{}, nil)
	in.SetRootView("app2.html")
	if in.RootView() != "app2.html" {
		t.Fatal("rootView should be app2.html")
	}
}

func TestInertia_Render(t *testing.T) {
	t.Run("render full html", func(t *testing.T) {
		e := echo.New()
		e.Renderer = NewRenderer(filepath.Join("testdata", "*.html"), nil)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		in := New(c, "app.html", map[string]interface{}{}, func() string {
			return "123456789"
		})

		err := in.Render(http.StatusOK, "Home", map[string]interface{}{
			"title": "Home Page title",
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := `<div id="app" data-page="{&#34;component&#34;:&#34;Home&#34;,&#34;props&#34;:{&#34;title&#34;:&#34;Home Page title&#34;},&#34;url&#34;:&#34;/&#34;,&#34;version&#34;:&#34;123456789&#34;}"></div>`
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s\nwant: %s", body, expected)
		}
	})

	t.Run("render JSON", func(t *testing.T) {
		e := echo.New()
		e.Renderer = NewRenderer(filepath.Join("testdata", "*.html"), nil)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(HeaderXInertia, "true")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		in := New(c, "app.html", map[string]interface{}{}, func() string {
			return "123456789"
		})

		err := in.Render(http.StatusOK, "Home", map[string]interface{}{
			"title": "Home Page title",
		})
		if err != nil {
			t.Fatal(err)
		}

		expected := `{"component":"Home","props":{"title":"Home Page title"},"url":"/","version":"123456789"}`
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s\nwant: %s", body, expected)
		}
	})
}

func TestInertia_RenderWithViewData(t *testing.T) {
	e := echo.New()
	e.Renderer = NewRenderer(filepath.Join("testdata", "*.html"), nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	in := New(c, "app_with_view_data.html", map[string]interface{}{}, func() string {
		return "123456789"
	})

	err := in.RenderWithViewData(http.StatusOK, "Home", map[string]interface{}{
		"title": "Home Page title",
	}, map[string]interface{}{
		"viewData": "This is view data",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := `<div>This is view data</div>
<div id="app" data-page="{&#34;component&#34;:&#34;Home&#34;,&#34;props&#34;:{&#34;title&#34;:&#34;Home Page title&#34;},&#34;url&#34;:&#34;/&#34;,&#34;version&#34;:&#34;123456789&#34;}"></div>`
	body := strings.TrimSuffix(rec.Body.String(), "\n")
	if body != expected {
		t.Errorf("unexpected body: %s\nwant: %s", body, expected)
	}
}
