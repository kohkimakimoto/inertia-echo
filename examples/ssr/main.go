package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
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

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	// setup inertia
	r := inertia.NewHTMLRenderer()
	r.Debug = IsDebug()
	r.MustParseGlob(filepath.Join(optDir, "views/*.html"))
	r.ViteBasePath = "/build"
	r.AddViteEntryPoint("assets/app.tsx")
	r.MustParseViteManifestFile(filepath.Join(optDir, "public/build/manifest.json"))
	r.SsrFallbackOnError = true
	r.SsrErrorReporter = func(ctx *inertia.RenderContext, err error) {
		e.Logger.Error("SSR failed; falling back to CSR",
			"component", ctx.Page.Component,
			"url", ctx.Page.URL,
			"error", err,
		)
	}
	// Use SSR engine for server-side rendering
	ssrGateway := inertia.NewSsrEngineHTTPGateway()
	if IsDebug() {
		ssrGateway.Endpoint = r.ViteDevServerURL + "/__inertia_ssr"
	}
	r.SsrEngine = ssrGateway

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c *echo.Context) error {
		return inertia.Render(c, "Index", map[string]any{
			"title":   "SSR example powered by inertia-echo",
			"message": "SSR example",
		})
	})
	e.GET("/about", func(c *echo.Context) error {
		return inertia.Render(c, "About", map[string]any{
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
				e.Logger.Error("the Vite subprocess returned an error", "error", err)
			}
		}()
	}

	if !IsDebug() {
		go func() {
			// Run the standalone production SSR server.
			if err := subprocess.Run(&subprocess.Config{
				Command:         "npm",
				Args:            []string{"run", "start-ssr"},
				Stdout:          os.Stdout,
				StdoutFormatter: subprocess.PrefixFormatter("[SSR] "),
				Stderr:          os.Stderr,
				StderrFormatter: subprocess.PrefixFormatter("[SSR] "),
				Dir:             optDir,
			}); err != nil {
				e.Logger.Error("the SSR subprocess returned an error", "error", err)
			}
		}()
	}

	if err := e.Start(":8080"); err != nil {
		e.Logger.Error("failed to start server", "error", err)
	}
}
