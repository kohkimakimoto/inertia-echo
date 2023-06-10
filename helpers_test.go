package inertia

import (
	"io"
	"testing"
)

type mockRenderer struct {
	render func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error
}

func (r *mockRenderer) Render(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
	return r.render(w, name, data, in)
}

func testNewMockRenderer(t *testing.T, renderFunc func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error) Renderer {
	t.Helper()

	return &mockRenderer{render: renderFunc}
}
