package main

import (
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

func main() {
	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.MustParseGlob("views/*.html")
	r.ViteBasePath = "/build"
	r.MustParseViteManifestFile("public/build/manifest.json")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", "public")

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
			"message": "Hello, World!",
		})
	})

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
