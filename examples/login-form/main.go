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
	"github.com/kohkimakimoto/inertia-echo/examples/login-form/resources"
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

	e.Use(middleware.Recover())
	e.Use(middleware.RequestLogger())

	r := inertia.NewHTMLRenderer()
	r.Debug = isDebug
	r.ViteBasePath = "/build"
	if !r.Debug {
		r.ViteManifest = inertia.MustParseViteManifestFS(resources.Public(), "build/manifest.json")
	}
	r.MustParseFS(resources.Views(), "*.html")

	e.Use(session.Middleware(session.NewCookieStore([]byte("secret"))))
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())
	e.Use(inertia.EncryptHistoryMiddleware())

	e.StaticFS("/", resources.Public())

	e.GET("/", func(c *echo.Context) error {
		s := session.MustGet(c)
		authEmail := s.GetString("auth_email")
		slog.Debug("authEmail", "email", authEmail)

		return inertia.Render(c, "Index", map[string]any{
			"message": "You are logged in!",
			"email":   authEmail,
		})
	}, AuthMiddleware)

	e.GET("/about", func(c *echo.Context) error {
		return inertia.Render(c, "About", map[string]any{
			"title": "About inertia-echo",
		})
	}, AuthMiddleware)

	e.GET("/login", func(c *echo.Context) error {
		s := session.MustGet(c)
		if authEmail := s.GetString("auth_email"); authEmail != "" {
			// Redirect to the home page if already logged in
			inertia.ClearHistory(c)
			return c.Redirect(http.StatusFound, "/")
		}
		return inertia.Render(c, "Login", map[string]any{})
	})

	type Form struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	e.POST("/login", func(c *echo.Context) error {
		form := &Form{}
		if err := c.Bind(form); err != nil {
			return err
		}

		if form.Email != "kohki.makimoto@gmail.com" {
			// display the login page again if the email is not correct
			return inertia.Render(c, "Login", map[string]any{
				"errors": map[string]string{
					"email": "Invalid email address",
				},
			})
		}

		// This is an example, so we are not checking the password.
		// Any input can be used as valid credentials.
		s := session.MustGet(c)
		s.Set("auth_email", form.Email)
		if err := s.Save(); err != nil {
			return err
		}
		slog.Debug("User authenticated", "email", form.Email)

		// Redirect to the home page after login
		inertia.ClearHistory(c)
		return c.Redirect(http.StatusFound, "/")
	})

	e.GET("/logout", func(c *echo.Context) error {
		s := session.MustGet(c)
		// Clear the session
		s.Clear()
		if err := s.Save(); err != nil {
			return err
		}
		slog.Debug("User logged out")

		// Redirect to the login page after logout
		inertia.ClearHistory(c)
		return c.Redirect(http.StatusFound, "/login")
	})

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

	if vite != nil {
		if err := vite.Wait(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("the Vite subprocess returned an error", "error", err)
		}
	}
}

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		s := session.MustGet(c)
		authEmail := s.GetString("auth_email")
		if authEmail == "" {
			slog.Debug("User is not authenticated, redirecting to login page")
			return c.Redirect(http.StatusFound, "/login")
		}
		return next(c)
	}
}
