package inertia

import (
	"os"
	"testing"
)

func TestDefaultVersionFunc(t *testing.T) {
	_ = os.Setenv("GAE_VERSION", "123456789")
	vf := defaultVersionFunc()
	version := vf()
	if version != "123456789" {
		t.Errorf("expected version to be %s, got %s", "123456789", version)
	}
}
