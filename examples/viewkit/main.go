package main

import (
	"flag"
	viewkit "github.com/kohkimakimoto/echo-viewkit"
	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/ext/viewkitext/v2"
	inertia "github.com/kohkimakimoto/inertia-echo/v2"
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

	// Echo ViewKit
	v := viewkit.New()
	v.Debug = IsDebug()
	v.BaseDir = filepath.Join(optDir, "views")
	v.Vite = true
	v.ViteDevMode = IsDebug()
	if !v.ViteDevMode {
		// vite production build config
		v.ViteManifest = viewkit.MustParseViteManifestFile("public/build/manifest.json")
		v.ViteBasePath = "/build"
	}
	r := viewkitext.NewRenderer(v.MustRenderer())
	r.SsrEngine = inertia.NewSsrEngineHTTPGateway()
	e.Renderer = r.Internal()

	// setup inertia
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]any{
			"title":   "Echo ViewKit integration example powered by inertia-echo",
			"message": "Echo ViewKit integration example ",
		})
	})

	e.GET("/about", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "About", map[string]any{
			"title": "About inertia-echo",
		})
	})

	if v.ViteDevMode {
		// Start Vite dev server if it's in dev mode
		go func() {
			if err := v.StartViteDevServer(); err != nil {
				e.Logger.Errorf("the vite dev server returned an error: %v", err)
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
