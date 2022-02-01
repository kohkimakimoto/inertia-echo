package inertia

import (
	"bytes"
	"html/template"
	"testing"
)

const testManifestFIle = `{
  "src/main.tsx": {
    "file": "assets/main.71a4fcb6.js",
    "src": "src/main.tsx",
    "isEntry": true,
    "imports": [
      "_vendor.9256646d.js"
    ]
  },
  "_vendor.9256646d.js": {
    "file": "assets/vendor.9256646d.js"
  }
}
`

func TestViteEntry(t *testing.T) {
	manifest, err := ParseViteManifest([]byte(testManifestFIle))
	if err != nil {
		t.Fatal(err)
	}

	tmpl := template.Must(template.New("template").Funcs(map[string]interface{}{
		"vite_entry": ViteEntry(manifest),
	}).Parse(`{{ $main := vite_entry "src/main.tsx" }}{{ $main.file }}`))

	buf := &bytes.Buffer{}
	if err = tmpl.ExecuteTemplate(buf, "template", nil); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if out != `assets/main.71a4fcb6.js` {
		t.Errorf("unexpected output: %s", out)
	}
}
