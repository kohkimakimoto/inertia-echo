package inertia

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestCSRF(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	csrf := CSRF()
	h := csrf(func(c echo.Context) error {
		return c.String(http.StatusOK, "test")
	})
	_ = h(c)

	cookie := rec.Header().Get(echo.HeaderSetCookie)
	if !strings.Contains(cookie, "XSRF-TOKEN") {
		t.Errorf("should contain XSRF-TOKEN, but not '%v'", cookie)
	}
}

func TestCSRFWithConfig(t *testing.T) {
	t.Run("has skipper but it should not skip", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		csrf := CSRFWithConfig(CSRFConfig{
			Skipper: func(c echo.Context) bool {
				path := c.Request().URL.Path
				return strings.HasPrefix(path, "/should_skip")
			},
		})
		h := csrf(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})
		_ = h(c)

		cookie := rec.Header().Get(echo.HeaderSetCookie)
		if !strings.Contains(cookie, "XSRF-TOKEN") {
			t.Errorf("should contain XSRF-TOKEN, but not '%v'", cookie)
		}
	})

	t.Run("has skipper and it should skip", func(t *testing.T) {
		e := echo.New()
		req := httptest.NewRequest(http.MethodGet, "/should_skip", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		csrf := CSRFWithConfig(CSRFConfig{
			Skipper: func(c echo.Context) bool {
				path := c.Request().URL.Path
				return strings.HasPrefix(path, "/should_skip")
			},
		})
		h := csrf(func(c echo.Context) error {
			return c.String(http.StatusOK, "test")
		})
		_ = h(c)

		cookie := rec.Header().Get(echo.HeaderSetCookie)
		if strings.Contains(cookie, "XSRF-TOKEN") {
			t.Errorf("should NOT contain XSRF-TOKEN because it should be skipped, but not '%v'", cookie)
		}
	})
}
