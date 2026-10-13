package inertia

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// Handler is a helper function that makes an inertia route without implementing handler function.
func Handler(component string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return Render(c, http.StatusOK, component, nil)
	}
}

func HandlerWithProps(component string, props map[string]any) echo.HandlerFunc {
	return func(c *echo.Context) error {
		return Render(c, http.StatusOK, component, props)
	}
}
