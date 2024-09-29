package inertia

import (
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInertia_SetRenderer(t *testing.T) {
	i := &Inertia{
		renderer: nil,
	}
	if i.Renderer() != nil {
		t.Errorf("renderer should be nil")
	}

	i.SetRenderer(testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
		return nil
	}))

	if i.Renderer() == nil {
		t.Errorf("renderer should not be nil")
	}
}

func TestInertia_IsSsrDisabled(t *testing.T) {
	i := &Inertia{
		isSsrDisabled: false,
	}
	i.EnableSsr()
	if i.IsSsrDisabled() {
		t.Errorf("ssr should not be disabled")
	}
	if !i.IsSsrEnabled() {
		t.Errorf("ssr should be enabled")
	}

	i.DisableSsr()
	if !i.IsSsrDisabled() {
		t.Errorf("ssr should be disabled")
	}
	if i.IsSsrEnabled() {
		t.Errorf("ssr should not be enabled")
	}
}

func TestInertia_SetRootView(t *testing.T) {
	i := &Inertia{
		rootView: "",
	}
	if i.RootView() != "" {
		t.Errorf("root view should be empty")
	}

	i.SetRootView("app.html")
	if i.RootView() != "app.html" {
		t.Errorf("root view should be app.html")
	}
}

func TestInertia_Share(t *testing.T) {
	i := &Inertia{
		sharedProps: map[string]interface{}{},
	}
	if len(i.Shared()) != 0 {
		t.Errorf("shared props should be empty")
	}

	i.Share(map[string]interface{}{
		"foo": "bar",
	})
	if len(i.Shared()) != 1 {
		t.Errorf("shared props should have 1 item")
	}
	if i.Shared()["foo"] != "bar" {
		t.Errorf("shared props should have foo=bar")
	}

	i.FlushShared()
	if len(i.Shared()) != 0 {
		t.Errorf("shared props should be empty")
	}
}

func TestInertia_Version(t *testing.T) {
	i := &Inertia{
		version: func() string {
			return "1.0.0"
		},
	}
	if i.Version() != "1.0.0" {
		t.Errorf("version should be 1.0.0")
	}

	i.SetVersion(func() string {
		return "2.0.0"
	})
	if i.Version() != "2.0.0" {
		t.Errorf("version should be 2.0.0")
	}
}

func TestInertia_Location(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(HeaderXInertia, "true")
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)

	i := &Inertia{
		c: c,
	}

	err := i.Location("/foo")
	if err != nil {
		t.Errorf("should not return error")
	}
	if c.Response().Header().Get(HeaderXInertiaLocation) != "/foo" {
		t.Errorf("%s should be /foo", HeaderXInertiaLocation)
	}
}

func TestInertia_Render(t *testing.T) {
	t.Run("should render with renderer", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		i := &Inertia{
			c:           c,
			rootView:    "app.html",
			sharedProps: map[string]interface{}{},
			version: func() string {
				return "1.0.0"
			},
			renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
				if name != "app.html" {
					t.Errorf("name should be app.html")
				}
				page := data["page"].(*Page)
				if page.Component != "Index" {
					t.Errorf("page component should be Index")
				}
				if page.Props["message"] != "Hello World" {
					t.Errorf("page props message should be Hello World")
				}
				return nil
			}),
		}

		err := i.Render(http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello World",
		})
		if err != nil {
			t.Errorf("should not return error")
		}
	})

	t.Run("should render with renderer and shared props", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		i := &Inertia{
			c:        c,
			rootView: "app.html",
			sharedProps: map[string]interface{}{
				"foo": "bar",
			},
			version: func() string {
				return "1.0.0"
			},
			renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
				if name != "app.html" {
					t.Errorf("name should be app.html")
				}
				page := data["page"].(*Page)
				if page.Component != "Index" {
					t.Errorf("page component should be Index")
				}
				if page.Props["message"] != "Hello World" {
					t.Errorf("page props message should be Hello World")
				}
				if page.Props["foo"] != "bar" {
					t.Errorf("page props foo should be bar")
				}
				return nil
			}),
		}

		err := i.Render(http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello World",
		})
		if err != nil {
			t.Errorf("should not return error")
		}
	})

	t.Run("does not include Lazy props", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		i := &Inertia{
			c:           c,
			rootView:    "app.html",
			sharedProps: map[string]interface{}{},
			version: func() string {
				return "1.0.0"
			},
			renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
				if name != "app.html" {
					t.Errorf("name should be app.html")
				}
				page := data["page"].(*Page)
				if page.Component != "Index" {
					t.Errorf("page component should be Index")
				}
				if page.Props["key1"] != "value1" {
					t.Errorf("page props key1 should be value1")
				}
				if page.Props["key2"] != "value2" {
					t.Errorf("page props key2 should be value2")
				}
				if page.Props["key3"] != nil {
					// Lazy props should not be evaluated
					t.Errorf("page props key3 should be nil")
				}
				return nil
			}),
		}

		err := i.Render(http.StatusOK, "Index", map[string]interface{}{
			"key1": "value1",
			"key2": func() interface{} {
				return "value2"
			},
			"key3": Lazy(func() (interface{}, error) {
				return "value3", nil
			}),
		})
		if err != nil {
			t.Errorf("should not return error")
		}
	})

	t.Run("include Lazy props with partial rendering", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderXInertiaPartialComponent, "Index")
		req.Header.Set(HeaderXInertiaPartialData, "key2,key3")
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)
		i := &Inertia{
			c:           c,
			rootView:    "app.html",
			sharedProps: map[string]interface{}{},
			version: func() string {
				return "1.0.0"
			},
			renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
				if name != "app.html" {
					t.Errorf("name should be app.html")
				}
				page := data["page"].(*Page)
				if page.Component != "Index" {
					t.Errorf("page component should be Index")
				}
				if page.Props["key1"] != nil {
					t.Errorf("page props key1 should be nil")
				}
				if page.Props["key2"] != "value2" {
					t.Errorf("page props key2 should be value2")
				}
				if page.Props["key3"] != "value3" {
					t.Errorf("page props key3 should be value3")
				}
				return nil
			}),
		}

		err := i.Render(http.StatusOK, "Index", map[string]interface{}{
			"key1": "value1",
			"key2": func() interface{} {
				return "value2"
			},
			"key3": Lazy(func() (interface{}, error) {
				return "value3", nil
			}),
		})
		if err != nil {
			t.Errorf("should not return error")
		}
	})

	t.Run("should respond with JSON", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderXInertia, "true")
		res := httptest.NewRecorder()
		c := e.NewContext(req, res)

		i := &Inertia{
			c:           c,
			rootView:    "app.html",
			sharedProps: map[string]interface{}{},
			version: func() string {
				return "1.0.0"
			},
			renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
				t.Errorf("should not call renderer")
				return nil
			}),
		}

		err := i.Render(http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello World",
		})
		if err != nil {
			t.Errorf("should not return error")
		}
		if res.Code != http.StatusOK {
			t.Errorf("should respond with status code 200")
		}
		if res.Header().Get("Content-Type") != "application/json" {
			t.Errorf("should respond with Content-Type application/json")
		}
	})
}

func TestInertia_RenderWithViewData(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)

	i := &Inertia{
		c:           c,
		rootView:    "app.html",
		sharedProps: map[string]interface{}{},
		version: func() string {
			return "1.0.0"
		},
		renderer: testNewMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
			if name != "app.html" {
				t.Errorf("name should be app.html")
			}
			page := data["page"].(*Page)
			if page.Component != "Index" {
				t.Errorf("page component should be Index")
			}
			if page.Props["message"] != "Hello World" {
				t.Errorf("page props message should be Hello World")
			}
			if data["key1"] != "value1" {
				t.Errorf("data key1 should be value1")
			}
			return nil
		}),
	}

	err := i.RenderWithViewData(http.StatusOK, "Index", map[string]interface{}{
		"message": "Hello World",
	}, map[string]interface{}{
		"key1": "value1",
	})
	if err != nil {
		t.Errorf("should not return error")
	}
}
