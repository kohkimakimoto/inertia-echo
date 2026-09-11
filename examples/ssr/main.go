package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	// setup inertia
	r := inertia.NewHTMLRenderer()
	r.Debug = IsDebug()
	r.ViteBasePath = "/build"
	if !r.Debug {
		r.ViteManifest = inertia.MustParseViteManifestFile(filepath.Join(optDir, "public/build/manifest.json"))
	}
	r.MustParseGlob(filepath.Join(optDir, "views/*.html"))
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

	var child *subprocess.Process
	if IsDebug() {
		p, err := subprocess.Start(ctx, subprocess.Config{
			Command:         "npm",
			Args:            []string{"run", "dev"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
			Dir:             optDir,
		})
		if err != nil {
			e.Logger.Error("failed to start Vite subprocess", "error", err)
			return
		}
		child = p
	} else {
		p, err := subprocess.Start(ctx, subprocess.Config{
			Command:         "npm",
			Args:            []string{"run", "start-ssr"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[SSR] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[SSR] "),
			Dir:             optDir,
		})
		if err != nil {
			e.Logger.Error("failed to start SSR subprocess", "error", err)
			return
		}
		child = p
	}

	if err := (echo.StartConfig{Address: ":8080"}).Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		e.Logger.Error("failed to start server", "error", err)
	}

	if child != nil {
		if err := child.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			e.Logger.Error("the subprocess returned an error", "error", err)
		}
	}
}
