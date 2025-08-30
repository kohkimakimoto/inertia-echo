package main

import (
	"net/http"

	inertia "github.com/kohkimakimoto/inertia-echo/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	r := inertia.NewHTMLRenderer()
	r.MustParseGlob("views/*.html")
	r.ViteBasePath = "/build"
	r.MustParseViteManifestFile("public/build/manifest.json")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "public")

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
