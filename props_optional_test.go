package inertia

import (
	"testing"
	"time"
)

func TestOptionalBuilder(t *testing.T) {
	t.Run("implements IgnoreFirstLoad", func(t *testing.T) {
		var _ IgnoreFirstLoadProp = Optional(func() (any, error) { return nil, nil })
	})

	t.Run("once modifiers", func(t *testing.T) {
		until := time.Date(2026, time.November, 1, 0, 0, 0, 0, time.UTC)
		p := Optional(func() (any, error) { return "x", nil }).
			Once().
			As("opt-key").
			Fresh(true).
			Until(until)
		got := p.onceSettings()
		if !got.enabled || got.key != "opt-key" || !got.fresh {
			t.Fatalf("settings=%+v", got)
		}
		if got.until == nil || !got.until.Equal(until) {
			t.Fatalf("until=%v", got.until)
		}

		p.For(3 * time.Second)
		got = p.onceSettings()
		if got.until != nil {
			t.Fatal("Until should be cleared by For")
		}
		if got.duration == nil || *got.duration != 3*time.Second {
			t.Fatalf("duration=%v", got.duration)
		}
	})
}
