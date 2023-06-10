package inertia

import (
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	t.Run("should render", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/about", nil)
		res := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, res)
		m := Middleware(testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
			page := data["page"].(*Page)
			if page.Component != "About" {
				t.Errorf("expected component: %s, got: %s", "About", page.Component)
			}
			return nil
		}))

		err := m(Handler("About"))(c)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("should render with props", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/about", nil)
		res := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, res)
		m := Middleware(testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
			page := data["page"].(*Page)
			if page.Component != "About" {
				t.Errorf("expected component: %s, got: %s", "About", page.Component)
			}
			if page.Props["title"] != "About Page" {
				t.Errorf("expected props: %s, got: %s", "About Page", page.Props["title"])
			}
			return nil
		}))

		err := m(Handler("About", map[string]interface{}{"title": "About Page"}))(c)
		if err != nil {
			t.Fatal(err)
		}
	})

}
