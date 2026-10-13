package inertia

import (
	"reflect"
	"testing"
	"time"
)

func TestDeferBuilders(t *testing.T) {
	t.Run("default group", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil })
		if p.Group() != "default" {
			t.Fatalf("group=%q", p.Group())
		}
	})

	t.Run("explicit group", func(t *testing.T) {
		p := DeferWithGroup(func() (any, error) { return nil, nil }, "attrs")
		if p.Group() != "attrs" {
			t.Fatalf("group=%q", p.Group())
		}
	})

	t.Run("implements IgnoreFirstLoad", func(t *testing.T) {
		var _ IgnoreFirstLoadProp = Defer(func() (any, error) { return nil, nil })
	})
}

func TestDeferMergeModifiers(t *testing.T) {
	t.Run("Merge enables root append", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil })
		p.Merge()
		if !p.ShouldMerge() || p.ShouldDeepMerge() {
			t.Fatalf("merge=%v deep=%v", p.ShouldMerge(), p.ShouldDeepMerge())
		}
		got := p.mergeSettings()
		if !got.appendRoot || got.deepMerge {
			t.Fatalf("settings=%+v", got)
		}
	})

	t.Run("DeepMerge", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil })
		p.DeepMerge()
		if !p.ShouldMerge() || !p.ShouldDeepMerge() {
			t.Fatal("expected merge and deepMerge")
		}
		got := p.mergeSettings()
		if !got.deepMerge || got.appendRoot {
			t.Fatalf("settings=%+v", got)
		}
	})

	t.Run("Append paths and MatchOn", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil }).
			Append("data").
			MatchOn("data.id")
		got := p.mergeSettings()
		if !reflect.DeepEqual(got.appendPaths, []string{"data"}) {
			t.Fatalf("appendPaths=%v", got.appendPaths)
		}
		if !reflect.DeepEqual(got.matchesOn, []string{"data.id"}) {
			t.Fatalf("matchesOn=%v", got.matchesOn)
		}
		if !reflect.DeepEqual(p.MatchesOn(), []string{"data.id"}) {
			t.Fatalf("MatchesOn=%v", p.MatchesOn())
		}
	})

	t.Run("Prepend root", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil }).Prepend()
		got := p.mergeSettings()
		if !got.prependRoot || got.deepMerge {
			t.Fatalf("settings=%+v", got)
		}
	})
}

func TestDeferOnceAndRescue(t *testing.T) {
	t.Run("once modifiers", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil }).
			Once().
			As("defer-key").
			Fresh(true).
			For(time.Minute)
		got := p.onceSettings()
		if !got.enabled || got.key != "defer-key" || !got.fresh {
			t.Fatalf("settings=%+v", got)
		}
		if got.duration == nil || *got.duration != time.Minute {
			t.Fatalf("duration=%v", got.duration)
		}
	})

	t.Run("Rescue without reporter", func(t *testing.T) {
		p := Defer(func() (any, error) { return nil, nil }).Rescue()
		if !p.rescue || p.reporter != nil {
			t.Fatalf("rescue=%v reporter=%v", p.rescue, p.reporter)
		}
	})

	t.Run("Rescue with reporter", func(t *testing.T) {
		var called bool
		p := Defer(func() (any, error) { return nil, nil }).Rescue(func(string, error) {
			called = true
		})
		if !p.rescue || p.reporter == nil {
			t.Fatal("expected rescue reporter")
		}
		p.reporter("path", nil)
		if !called {
			t.Fatal("reporter was not invoked")
		}
	})
}
