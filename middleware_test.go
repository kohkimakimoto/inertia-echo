package inertia

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestMiddleware(t *testing.T) {
	t.Run("just run inertia middleware", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		m := Middleware()

		err := m(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		if err != nil {
			t.Fatal(err)
		}

		if rec.Body.String() != "test" {
			t.Errorf("expected body to be 'test', got %s", rec.Body.String())
		}
	})

	t.Run("detect version change", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/example", nil)
		req.Header.Add(HeaderXInertia, "true")
		req.Header.Add(HeaderXInertiaVersion, "1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		m := MiddlewareWithConfig(MiddlewareConfig{
			VersionFunc: func() string {
				// return updated version
				return "2"
			},
		})

		err := m(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})(c)

		if err != nil {
			t.Fatal(err)
		}

		status := rec.Result().StatusCode
		if status != http.StatusConflict {
			t.Errorf("expected status code to be %d, got %d", http.StatusConflict, status)
		}

		url := rec.Result().Header.Get(HeaderXInertiaLocation)
		if url != "/example" {
			t.Errorf("expected header X-Inertia-Location to be %s, got %s", "/example", url)
		}
	})
}

func TestMiddlewareAndRender(t *testing.T) {
	t.Run("render with shared data", func(t *testing.T) {
		e := echo.New()
		e.Renderer = NewRenderer("./testdata/*.html", map[string]interface{}{})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(HeaderXInertia, "true")
		req.Header.Add(HeaderXInertiaVersion, "1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		m := MiddlewareWithConfig(MiddlewareConfig{
			VersionFunc: func() string {
				return "1"
			},
			// set shared value in the middleware
			Share: func(c echo.Context) (map[string]interface{}, error) {
				return map[string]interface{}{
					"key1": "value1",
				}, nil
			},
		})

		err := m(func(c echo.Context) error {
			i := MustGet(c)

			// set shared data by Inertia instance
			i.Share(map[string]interface{}{
				"key2": "value2",
			})

			// check shared data
			shared := i.Shared()
			if shared["key1"] != "value1" {
				t.Errorf("expected shared data to be %s, got %s", "value1", shared["key1"])
			}
			if shared["key2"] != "value2" {
				t.Errorf("expected shared data to be %s, got %s", "value2", shared["key2"])
			}

			return i.Render(http.StatusOK, "Page", map[string]interface{}{})
		})(c)

		if err != nil {
			t.Fatal(err)
		}

		expected := `{"component":"Page","props":{"key1":"value1","key2":"value2"},"url":"/","version":"1"}`
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s", body)
		}
	})

	t.Run("render after flushing shared data", func(t *testing.T) {
		e := echo.New()
		e.Renderer = NewRenderer("./testdata/*.html", map[string]interface{}{})
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Add(HeaderXInertia, "true")
		req.Header.Add(HeaderXInertiaVersion, "1")
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		m := MiddlewareWithConfig(MiddlewareConfig{
			VersionFunc: func() string {
				return "1"
			},
			// set shared value in the middleware
			Share: func(c echo.Context) (map[string]interface{}, error) {
				return map[string]interface{}{
					"key1": "value1",
				}, nil
			},
		})

		err := m(func(c echo.Context) error {
			i := MustGet(c)
			// flush shared value. it means that shared value will be cleared.
			i.FlushShared()
			return i.Render(http.StatusOK, "Page", map[string]interface{}{})
		})(c)

		if err != nil {
			t.Fatal(err)
		}

		expected := `{"component":"Page","props":{},"url":"/","version":"1"}`
		body := strings.TrimSuffix(rec.Body.String(), "\n")
		if body != expected {
			t.Errorf("unexpected body: %s", body)
		}
	})
}

func TestDefaultVersionFunc(t *testing.T) {
	_ = os.Setenv("GAE_VERSION", "123456789")
	vf := defaultVersionFunc()
	version := vf()
	if version != "123456789" {
		t.Errorf("expected version to be %s, got %s", "123456789", version)
	}
}

func TestMustGet(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	m := Middleware()

	err := m(func(c echo.Context) error {
		i := MustGet(c)
		if i == nil {
			t.Error("expected Inertia to be set")
		}
		return c.String(http.StatusOK, "test")
	})(c)
	if err != nil {
		t.Fatal(err)
	}
}

func TestHas(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	m := Middleware()

	err := m(func(c echo.Context) error {
		if !Has(c) {
			t.Errorf("expected Inertia exists")
		}
		return c.String(http.StatusOK, "test")
	})(c)
	if err != nil {
		t.Fatal(err)
	}
}
