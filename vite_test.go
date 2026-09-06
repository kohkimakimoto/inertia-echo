package inertia

import (
	"testing"
	"testing/fstest"
)

func TestParseViteManifest(t *testing.T) {
	data := []byte(`{
		"assets/app.tsx": {
			"file": "assets/app-abc123.js",
			"css": ["assets/app-def456.css"]
		}
	}`)

	m, err := ParseViteManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	chunk, ok := m["assets/app.tsx"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected chunk type: %#v", m["assets/app.tsx"])
	}
	if chunk["file"] != "assets/app-abc123.js" {
		t.Fatalf("unexpected file: %v", chunk["file"])
	}
}

func TestParseViteManifestFS(t *testing.T) {
	fsys := fstest.MapFS{
		"public/build/manifest.json": &fstest.MapFile{
			Data: []byte(`{"entry.js":{"file":"entry-hash.js"}}`),
		},
	}
	m, err := ParseViteManifestFS(fsys, "public/build/manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := m["entry.js"]; !ok {
		t.Fatalf("missing entry: %#v", m)
	}
}

func TestMustParseViteManifestPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic")
		}
	}()
	_ = MustParseViteManifest([]byte(`not-json`))
}
