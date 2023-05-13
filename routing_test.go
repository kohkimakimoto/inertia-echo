package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
)

func TestHandler(t *testing.T) {
	// TODO:
	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rec := httptest.NewRecorder()

	e := echo.New()
	e.Renderer = NewRenderer().MustParseGlob("testdata/*.html")
	e.GET("/about", Handler("About"))
	e.Use(Middleware())
	e.ServeHTTP(rec, req)

	if http.StatusOK != rec.Code {
		t.Errorf("expected status code to be %d, got %d", http.StatusOK, rec.Code)
	}
}
