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

	session "github.com/kohkimakimoto/echo-session/v5"
	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/examples/notes/internal/store"
	"github.com/kohkimakimoto/inertia-echo/examples/notes/resources"
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
	defaultErrorHandler := e.HTTPErrorHandler
	e.HTTPErrorHandler = func(c *echo.Context, err error) {
		var status echo.HTTPStatusCoder
		if errors.As(err, &status) && status.StatusCode() == http.StatusNotFound && inertia.Has(c) {
			if renderErr := notFound(c); renderErr == nil {
				return
			}
		}
		defaultErrorHandler(c, err)
	}
	db, err := store.OpenWithSeed("data", "seed")
	if err != nil {
		panic(err)
	}

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.Debug = isDebug
	r.ViteBasePath = "/build"
	if !r.Debug {
		r.ViteManifest = inertia.MustParseViteManifestFS(resources.Public(), "build/manifest.json")
	}
	r.MustParseFS(resources.Views(), "*.html")

	cookieStore := session.NewCookieStore([]byte("notes-example-secret"))
	// The example is served over plain HTTP at localhost:8080 in both modes.
	cookieStore.Options.Secure = false
	cookieStore.Options.SameSite = http.SameSiteLaxMode
	e.Use(session.Middleware(cookieStore))
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
		Share: func(c *echo.Context) (map[string]any, error) {
			var auth any
			theme := "system"
			if user, ok := db.GetUser(session.MustGet(c).GetString("user_id")); ok {
				auth = map[string]string{"id": user.ID, "name": user.Name, "email": user.Email}
				theme = db.GetSettings(user.ID).Theme
			}
			return map[string]any{"auth": auth, "theme": theme}, nil
		},
		ShareOnce: func(c *echo.Context) (map[string]any, error) {
			return map[string]any{"appName": inertia.Once(func() (any, error) { return "Notes", nil }).As("app-name")}, nil
		},
		RescueReporter: func(path string, err error) { slog.Error("deferred prop rescued", "path", path, "error", err) },
	}))
	e.Use(inertia.CSRF())
	e.Use(inertia.EncryptHistoryMiddleware())

	e.StaticFS("/", resources.Public())

	registerRoutes(e, db)
	var vite *subprocess.Process
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
		vite = p
	}

	if err := (echo.StartConfig{Address: ":8080"}).Start(ctx, e); err != nil && !errors.Is(err, http.ErrServerClosed) {
		slog.Error("failed to start server", "error", err)
	}
	stop()

	if vite != nil {
		if err := vite.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("the Vite subprocess returned an error", "error", err)
		}
	}
}
