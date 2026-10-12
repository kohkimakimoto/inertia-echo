package inertia

import (
	"encoding/json"
	"io/fs"
	"os"
)

// ViteManifest is the Go representation of Vite's manifest.json.
type ViteManifest map[string]any

// ParseViteManifest parses Vite manifest JSON bytes.
func ParseViteManifest(data []byte) (ViteManifest, error) {
	var manifest ViteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	return manifest, nil
}

// MustParseViteManifest is like ParseViteManifest but panics on error.
func MustParseViteManifest(data []byte) ViteManifest {
	m, err := ParseViteManifest(data)
	if err != nil {
		panic(err)
	}
	return m
}

// ParseViteManifestFile reads and parses a Vite manifest.json file.
func ParseViteManifestFile(name string) (ViteManifest, error) {
	b, err := os.ReadFile(name)
	if err != nil {
		return nil, err
	}
	return ParseViteManifest(b)
}

// MustParseViteManifestFile is like ParseViteManifestFile but panics on error.
func MustParseViteManifestFile(name string) ViteManifest {
	m, err := ParseViteManifestFile(name)
	if err != nil {
		panic(err)
	}
	return m
}

// ParseViteManifestFS reads and parses a Vite manifest.json from an fs.FS.
func ParseViteManifestFS(f fs.FS, name string) (ViteManifest, error) {
	b, err := fs.ReadFile(f, name)
	if err != nil {
		return nil, err
	}
	return ParseViteManifest(b)
}

// MustParseViteManifestFS is like ParseViteManifestFS but panics on error.
func MustParseViteManifestFS(f fs.FS, name string) ViteManifest {
	m, err := ParseViteManifestFS(f, name)
	if err != nil {
		panic(err)
	}
	return m
}
