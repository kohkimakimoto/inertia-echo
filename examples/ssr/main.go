package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/examples/ssr/resources"
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
	"github.com/labstack/echo/v5/middleware"
)

// BuildMode is overwritten at build time via -ldflags "-X main.BuildMode=production".
var BuildMode = "debug"

func main() {
	isDebug := BuildMode == "debug"

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	if isDebug {
		resources.UseDir(filepath.Join(root, "resources"))
	}

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.Debug = isDebug
	r.ViteBasePath = "/build"
	if !r.Debug {
		r.ViteManifest = inertia.MustParseViteManifestFS(resources.Public(), "build/manifest.json")
	}
	r.MustParseFS(resources.Views(), "*.html")
	r.SsrFallbackOnError = true
	r.SsrErrorReporter = func(ctx *inertia.RenderContext, err error) {
		slog.Error("SSR failed; falling back to CSR",
			"component", ctx.Page.Component,
			"url", ctx.Page.URL,
			"error", err,
		)
	}
	// Use SSR engine for server-side rendering
	ssrGateway := inertia.NewSsrEngineHTTPGateway()
	if isDebug {
		ssrGateway.Endpoint = r.ViteDevServerURL + "/__inertia_ssr"
	}
	r.SsrEngine = ssrGateway

	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())

	e.StaticFS("/", resources.Public())

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
	if isDebug {
		p, err := subprocess.Start(ctx, subprocess.Config{
			Command:         "npx",
			Args:            []string{"vite"},
			Stdout:          os.Stdout,
			StdoutFormatter: subprocess.PrefixFormatter("[Vite] "),
			Stderr:          os.Stderr,
			StderrFormatter: subprocess.PrefixFormatter("[Vite] "),
			Dir:             root,
		})
		if err != nil {
			slog.Error("failed to start Vite subprocess", "error", err)
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
			Dir:             root,
		})
		if err != nil {
			slog.Error("failed to start SSR subprocess", "error", err)
			return
		}
		child = p
	}

	if err := (echo.StartConfig{Address: ":8080"}).Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}

	if child != nil {
		if err := child.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("the subprocess returned an error", "error", err)
		}
	}
}
