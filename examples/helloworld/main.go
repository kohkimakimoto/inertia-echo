package main

import (
	"embed"
	"flag"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/kohkimakimoto/inertia-echo"
)

var Inertia = inertia.MustGet

func main() {
	var optDev bool
	flag.BoolVar(&optDev, "development", false, "Run dev mode")
	flag.Parse()

	e := echo.New()
	e.HTTPErrorHandler = func(err error, c echo.Context) {
		c.Logger().Error(err)
		e.DefaultHTTPErrorHandler(err, c)
	}
	e.Renderer = inertia.NewRendererWithFS(viewsFs, "views/*.html", map[string]interface{}{
		"vite_entry": inertia.ViteEntry(getViteManifest()),
		"is_dev": func() bool {
			return optDev
		},
	})

	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(inertia.Middleware())
	e.Use(inertia.CSRF())

	e.GET("/assets/*", echo.WrapHandler(http.StripPrefix("/assets/", http.FileServer(getAssetsFileSystem()))))
	e.GET("/", func(c echo.Context) error {
		return inertia.MustGet(c).Render(http.StatusOK, "Index", map[string]interface{}{
			"message": "Hello, World!",
		})
	})
	e.GET("/about", inertia.Handler("About"))

	e.Logger.Fatal(e.Start(":8080"))
}

//go:embed views
var viewsFs embed.FS

//go:embed gen
var genFs embed.FS

func getViteManifest() inertia.ViteManifest {
	b, err := genFs.ReadFile("gen/dist/manifest.json")
	if err != nil {
		panic(err)
	}
	manifest, err := inertia.ParseViteManifest(b)
	if err != nil {
		panic(err)
	}
	return manifest
}

func getAssetsFileSystem() http.FileSystem {
	fsys, err := fs.Sub(genFs, "gen/dist/assets")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}
