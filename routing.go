package inertia

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Handler is a helper function that makes an inertia route without implementing handler function.
func Handler(component string, props ...map[string]interface{}) echo.HandlerFunc {
	mergedProps := mergeProps(props...)
	return func(c echo.Context) error {
		return MustGet(c).Render(http.StatusOK, component, mergedProps)
	}
}
