package inertia

import (
	"testing"
)

type mockRenderer struct {
	render func(ctx *RenderContext) error
}

func (r *mockRenderer) Render(ctx *RenderContext) error {
	return r.render(ctx)
}

func testNewMockRenderer(t *testing.T, renderFunc func(ctx *RenderContext) error) *mockRenderer {
	t.Helper()

	return &mockRenderer{render: renderFunc}
}

// Helper function for deep equality comparison
func testDeepEqual(t *testing.T, a, b any) bool {
	t.Helper()

	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	switch va := a.(type) {
	case map[string]any:
		vb, ok := b.(map[string]any)
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for k, v := range va {
			if !testDeepEqual(t, v, vb[k]) {
				return false
			}
		}
		return true
	case []string:
		vb, ok := b.([]string)
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i, v := range va {
			if v != vb[i] {
				return false
			}
		}
		return true
	case []int:
		vb, ok := b.([]int)
		if !ok {
			return false
		}
		if len(va) != len(vb) {
			return false
		}
		for i, v := range va {
			if v != vb[i] {
				return false
			}
		}
		return true
	default:
		return a == b
	}
}
