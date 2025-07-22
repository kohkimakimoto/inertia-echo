package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	m := MiddlewareWithConfig(MiddlewareConfig{
		Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
			if ctx.Page.Component != "About" {
				t.Errorf("expected component: %s, got: %s", "About", ctx.Page.Component)
			}
			// When no props are provided, should default to empty map
			if len(ctx.Page.Props) != 0 {
				t.Errorf("expected empty props, got: %v", ctx.Page.Props)
			}
			return nil
		}),
	})

	err := m(Handler("About"))(c)
	if err != nil {
		t.Fatal(err)
	}

}

func TestHandlerWithProps(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()
	e := echo.New()
	c := e.NewContext(req, rec)
	m := MiddlewareWithConfig(MiddlewareConfig{
		Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
			if ctx.Page.Component != "About" {
				t.Errorf("expected component: %s, got: %s", "About", ctx.Page.Component)
			}
			if ctx.Page.Props["title"] != "About Page" {
				t.Errorf("expected props title: %s, got: %v", "About Page", ctx.Page.Props["title"])
			}
			if ctx.Page.Props["subtitle"] != "This is about page" {
				t.Errorf("expected props subtitle: %s, got: %v", "This is about page", ctx.Page.Props["subtitle"])
			}
			return nil
		}),
	})

	err := m(HandlerWithProps("About", map[string]any{
		"title":    "About Page",
		"subtitle": "This is about page",
	}))(c)
	if err != nil {
		t.Fatal(err)
	}
}
