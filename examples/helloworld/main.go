package main

import (
	"os"
	"path/filepath"

	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/examples/helloworld/resources"
	"github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// BuildMode is overwritten at build time via -ldflags "-X main.BuildMode=production".
var BuildMode = "debug"

func main() {
	isDebug := BuildMode == "debug"

	var root string
	if isDebug {
		var err error
		root, err = os.Getwd()
		if err != nil {
			panic(err)
		}
		resources.UseDir(filepath.Join(root, "resources"))
	}

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	// setup inertia
	r := inertia.NewHTMLRenderer()
	r.Debug = isDebug
	r.ViteBasePath = "/build"
	if !r.Debug {
		r.ViteManifest = inertia.MustParseViteManifestFS(resources.Public(), "build/manifest.json")
	}
	r.MustParseFS(resources.Views(), "*.html")

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.StaticFS("/", resources.Public())

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
			"title":   "Hello, World! powered by inertia-echo",
			"message": "Hello, World!",
		})
	})

	e.GET("/about", func(c *echo.Context) error {
		return inertia.Render(c, "About", map[string]any{
			"title": "About inertia-echo",
			"deferredMessage": inertia.Defer(func() (any, error) {
				return "Hello, World! from deferred props", nil
			}),
		})
	})

	if isDebug {
		go func() {
			// Run a subprocess for Vite development server.
			if err := subprocess.Run(subprocess.Config{
				Command:         "npm",
				Args:            []string{"run", "dev"},
				Stdout:          os.Stdout,
				StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
				Stderr:          os.Stderr,
				StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
				Dir:             root,
			}); err != nil {
				e.Logger.Error("the Vite subprocess returned an error", "error", err)
			}
		}()
	}

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
