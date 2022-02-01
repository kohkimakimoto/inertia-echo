package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestNewResponseWriterWrapper(t *testing.T) {
	t.Run("should buffer status code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, rec)
		var w *ResponseWriterWrapper
		err := func(c echo.Context) error {
			res := c.Response()
			w = NewResponseWriterWrapper(res.Writer)
			res.Writer = w
			return c.Redirect(http.StatusFound, "/example")
		}(c)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusOK {
			// status code 200, because the actual status code is buffered and not sent.
			t.Errorf("expected status code to be 200, got %d", rec.Code)
		}

		w.FlushHeader()
		if rec.Code != http.StatusFound {
			// you will get buffered status code after FlushHeader.
			t.Errorf("expected status code to be 302, got %d", rec.Code)
		}
	})

	t.Run("should NOT buffer status code", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		e := echo.New()
		c := e.NewContext(req, rec)
		var w *ResponseWriterWrapper
		err := func(c echo.Context) error {
			res := c.Response()
			w = NewResponseWriterWrapper(res.Writer)
			res.Writer = w
			return c.String(http.StatusNotFound, "/example")
		}(c)
		if err != nil {
			t.Fatal(err)
		}

		if rec.Code != http.StatusNotFound {
			// status code 404, 404 is not buffered.
			t.Errorf("expected status code to be 404, got %d", rec.Code)
		}

		w.FlushHeader()
		if rec.Code != http.StatusNotFound {
			// no effect by FlushHeader
			t.Errorf("expected status code to be 404, got %d", rec.Code)
		}
	})
}
