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
		t.Errorf("should cotain XSRF-TOKEN, but not '%v'", cookie)
	}
}
