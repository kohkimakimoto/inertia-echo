package main

import (
	"flag"
	"github.com/kohkimakimoto/inertia-echo"
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
	r := inertia.NewRenderer()
	r.Debug = e.Debug
	r.MustParseGlob(filepath.Join(optDir, "views/*.html"))
	r.ViteBasePath = "/dist/"
	r.AddViteEntryPoint("js/app.tsx")
	r.MustParseViteManifestFile(filepath.Join(optDir, "public/dist/manifest.json"))

	e.Use(inertia.Middleware(r))
	e.Use(inertia.CSRF())

	e.Static("/", filepath.Join(optDir, "public"))

	e.GET("/", func(c echo.Context) error {
		return inertia.Render(c, http.StatusOK, "Index", map[string]interface{}{
			"title":   "Hello, World! powered by inertia-echo",
			"message": "Hello, World!",
		})
	})

	e.Logger.Fatal(e.Start(":8080"))
}
