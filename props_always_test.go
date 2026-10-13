package inertia

import "testing"

func TestAlwaysBuilder(t *testing.T) {
	p := Always("keep")
	if p.value != "keep" {
		t.Fatalf("value=%v", p.value)
	}
}

func TestLazyBuilder(t *testing.T) {
	p := Lazy(func() (any, error) { return "lazy", nil })
	var _ IgnoreFirstLoadProp = p
	got, err := p.callback()
	if err != nil || got != "lazy" {
		t.Fatalf("callback got=%v err=%v", got, err)
	}
}
