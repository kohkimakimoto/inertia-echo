package resources

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/labstack/echo/v5"
)

//go:embed all:public
var publicFS embed.FS

//go:embed all:views
var viewsFS embed.FS

// FS resolves views and public assets from either the filesystem (debug)
// or embedded files (production).
type FS struct {
	debug bool
	dir   string // resources directory; used only when debug
}

// std is the package-level default, analogous to log's standard logger.
// The default mode is production (embedded files).
var std = &FS{}

// UseEmbed switches the package-level FS to embedded (production) mode.
func UseEmbed() { std.UseEmbed() }

// UseDir switches the package-level FS to filesystem (debug) mode.
// dir is the resources directory that contains views/ and public/.
func UseDir(dir string) { std.UseDir(dir) }

// Views returns the package-level views filesystem.
func Views() fs.FS { return std.Views() }

// Public returns the package-level public filesystem.
func Public() fs.FS { return std.Public() }

// UseEmbed switches this FS to embedded (production) mode.
func (f *FS) UseEmbed() {
	f.debug = false
	f.dir = ""
}

// UseDir switches this FS to filesystem (debug) mode.
// dir is the resources directory that contains views/ and public/.
func (f *FS) UseDir(dir string) {
	if dir == "" {
		panic("resources: dir must not be empty")
	}
	f.debug = true
	f.dir = dir
}

// Views returns an fs.FS rooted at the views directory.
func (f *FS) Views() fs.FS {
	if f.debug {
		return os.DirFS(filepath.Join(f.dir, "views"))
	}
	return echo.MustSubFS(viewsFS, "views")
}

// Public returns an fs.FS rooted at the public directory.
func (f *FS) Public() fs.FS {
	if f.debug {
		return os.DirFS(filepath.Join(f.dir, "public"))
	}
	return echo.MustSubFS(publicFS, "public")
}
