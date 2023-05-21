// package vite implements integration with Vite.js.
// This package is still experimental.

package vite

import (
	"encoding/json"
	"fmt"
	"github.com/kohkimakimoto/inertia-echo"
	"html/template"
	"io/fs"
	"os"
	"regexp"
	"strings"
)

type Vite struct {
	DevServerURL string
	Debug        bool
	Manifest     Manifest
	BasePath     string
	EntryPoints  []string
}

func New() *Vite {
	return &Vite{
		DevServerURL: "http://localhost:5173",
		Debug:        false,
		Manifest:     nil,
		BasePath:     "",
		EntryPoints:  []string{},
	}
}

func (v *Vite) NewRenderer() *inertia.Renderer {
	return inertia.NewRenderer().Funcs(v.FuncMap())
}

func (v *Vite) AddEntryPoint(entryPoint string) {
	v.EntryPoints = append(v.EntryPoints, entryPoint)
}

func (v *Vite) FuncMap() template.FuncMap {
	return template.FuncMap{
		"vite_react_refresh": v.fnReactRefresh,
		"vite":               v.fnVite,
	}
}

func (v *Vite) fnReactRefresh() template.HTML {
	if !v.Debug {
		return ""
	}

	return template.HTML(fmt.Sprintf(`<script type="module">
  import RefreshRuntime from '%s/@react-refresh'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>`, v.DevServerURL))
}

func (v *Vite) fnVite(entryPoints ...string) template.HTML {
	if len(entryPoints) == 0 {
		entryPoints = v.EntryPoints
	}

	if v.Debug {
		tags := []string{
			fmt.Sprintf(`<script type="module" src="%s/@vite/client"></script>`, v.DevServerURL),
		}
		for _, entryPoint := range entryPoints {
			tags = append(tags, v.genTag(fmt.Sprintf("%s/%s", v.DevServerURL, entryPoint)))
		}
		return template.HTML(strings.Join(tags, ""))
	}

	if v.Manifest == nil {
		panic("manifest is not loaded")
	}

	tags := []string{}
	for _, entryPoint := range entryPoints {
		chunk, ok := v.Manifest[entryPoint]
		if !ok {
			panic(fmt.Sprintf("unable to locate file in Vite manifest: %s", entryPoint))
		}

		if chunk, ok := chunk.(map[string]interface{}); ok {
			file := chunk["file"].(string)
			tags = append(tags, v.genTag(fmt.Sprintf("%s/%s", v.BasePath, file)))

			if css, ok := chunk["css"].([]interface{}); ok {
				for _, cssFile := range css {
					tags = append(tags, v.genTag(fmt.Sprintf("%s/%s", v.BasePath, cssFile)))
				}
			}
		}
	}
	return template.HTML(strings.Join(tags, ""))
}

func (v *Vite) genTag(path string) string {
	if isCssPath(path) {
		return fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, path)
	} else {
		return fmt.Sprintf(`<script type="module" src="%s"></script>`, path)
	}
}

var cssRe = regexp.MustCompile(`\.(css|less|sass|scss|styl|stylus|pcss|postcss)$`)

func isCssPath(name string) bool {
	return cssRe.MatchString(name)
}

func (v *Vite) ParseManifest(data []byte) error {
	m, err := ParseManifest(data)
	if err != nil {
		return err
	}
	v.Manifest = m
	return nil
}

func (v *Vite) MustParseManifest(data []byte) {
	if err := v.ParseManifest(data); err != nil {
		panic(err)
	}
}

func (v *Vite) ParseManifestFile(name string) error {
	m, err := ParseManifestFile(name)
	if err != nil {
		return err
	}
	v.Manifest = m
	return nil
}

func (v *Vite) MustParseManifestFile(name string) {
	if err := v.ParseManifestFile(name); err != nil {
		panic(err)
	}
}

func (v *Vite) ParseManifestFS(f fs.FS, name string) error {
	m, err := ParseManifestFS(f, name)
	if err != nil {
		return err
	}
	v.Manifest = m
	return nil
}

func (v *Vite) MustParseManifestFS(f fs.FS, name string) {
	if err := v.ParseManifestFS(f, name); err != nil {
		panic(err)
	}
}

type Manifest map[string]interface{}

func ParseManifest(data []byte) (Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

func ParseManifestFile(name string) (Manifest, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return ParseManifest(b)
}

func ParseManifestFS(f fs.FS, name string) (Manifest, error) {
	b, err := fs.ReadFile(f, name)
	if err != nil {
		return nil, err
	}
	return ParseManifest(b)
}
