package main

import (
	"flag"
	"github.com/kohkimakimoto/inertia-echo"
	"github.com/kohkimakimoto/inertia-echo/vite"
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

	// initialize vite integration for inertia
	v := vite.New()
	v.Debug = e.Debug
	v.BasePath = "/dist"
	v.AddEntryPoint("js/app.tsx")
	if !v.Debug {
		v.MustParseManifestFile(filepath.Join(optDir, "public/dist/manifest.json"))
	}

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	// setup inertia
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{
		Renderer: v.NewRenderer().MustParseGlob(filepath.Join(optDir, "views/*.html")),
	}))
	e.Use(inertia.CSRF())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello, World!",
		})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
