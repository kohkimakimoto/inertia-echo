package main

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"

	session "github.com/kohkimakimoto/echo-session"
	"github.com/kohkimakimoto/go-subprocess"
	"github.com/kohkimakimoto/inertia-echo/v2"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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

	e.Use(session.Middleware(session.NewCookieStore([]byte("secret"))))
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: r,
	}))
	e.Use(inertia.CSRF())
	e.Use(inertia.EncryptHistoryMiddleware())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c echo.Context) error {
		s := session.MustGet(c)
		authEmail := s.GetString("auth_email")
		c.Logger().Debugf("authEmail: %v", authEmail)

		return inertia.Render(c, "Index", map[string]any{
			"message": "You are logged in!",
			"email":   authEmail,
		})
	}, AuthMiddleware)

	e.GET("/about", func(c echo.Context) error {
		return inertia.Render(c, "About", map[string]any{
			"title": "About inertia-echo",
		})
	}, AuthMiddleware)

	e.GET("/login", func(c echo.Context) error {
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

	e.POST("login", func(c echo.Context) error {
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
		c.Logger().Debugf("User authenticated: %s", form.Email)

		// Redirect to the home page after login
		inertia.ClearHistory(c)
		return c.Redirect(http.StatusFound, "/")
	})

	e.GET("/logout", func(c echo.Context) error {
		s := session.MustGet(c)
		// Clear the session
		s.Clear()
		if err := s.Save(); err != nil {
			return err
		}
		c.Logger().Debug("User logged out")

		// Redirect to the login page after logout
		inertia.ClearHistory(c)
		return c.Redirect(http.StatusFound, "/login")
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

	e.Logger.Fatal(e.Start(":8080"))
}

func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		s := session.MustGet(c)
		authEmail := s.GetString("auth_email")
		if authEmail == "" {
			c.Logger().Debug("User is not authenticated, redirecting to login page")
			return c.Redirect(http.StatusFound, "/login")
		}
		return next(c)
	}
}
