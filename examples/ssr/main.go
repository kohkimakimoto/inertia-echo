package main

import (
	"flag"
	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"net/http"
	"os"
	"path/filepath"
)

var BuildMode = "debug"

func IsDebug() bool {
	return BuildMode == "debug"
}

func main() {
	var optDir string
	flag.StringVar(&optDir, "dir", "", "project directory")
	flag.Parse()

	if optDir == "" {
		optDir, _ = os.Getwd()
	}

	e := echo.New()
	e.Debug = IsDebug()

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	// setup inertia
	r := inertia.NewHTMLRenderer()
	r.Debug = e.Debug
	r.MustParseGlob(filepath.Join(optDir, "views/*.html"))
	r.ViteBasePath = "/build"
	r.AddViteEntryPoint("assets/app.tsx")
	r.MustParseViteManifestFile(filepath.Join(optDir, "public/build/manifest.json"))
	// Use SSR engine for server-side rendering
	r.SsrEngine = inertia.NewSsrEngineHTTPGateway()

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]interface{}{
			"title":   "SSR example powered by inertia-echo",
			"message": "SSR example",
		})
	})
	e.GET("/about", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "About", map[string]any{
			"title": "About inertia-echo",
		})
	})

	if IsDebug() {
		go func() {
			// Run a subprocess for Vite development server.
			if err := subprocess.Run(&subprocess.Config{
				Command:         "npm",
				Args:            []string{"run", "dev"},
				Stdout:          os.Stdout,
				StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
				Stderr:          os.Stderr,
				StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
				Dir:             optDir,
			}); err != nil {
				e.Logger.Errorf("the Vite subprocess returned an error: %v", err)
			}
		}()
	}

	go func() {
		// Run SSR server.
		if err := subprocess.Run(&subprocess.Config{
			Command:         "npm",
			Args:            []string{"run", "start-ssr"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[SSR] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[SSR] "),
			Dir:             optDir,
		}); err != nil {
			e.Logger.Errorf("the SSR subprocess returned an error: %v", err)
		}
	}()

	e.Logger.Fatal(e.Start(":8080"))
}
