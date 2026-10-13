package inertia

import (
	"reflect"
	"testing"
	"time"
)

func TestMergeBuilders(t *testing.T) {
	t.Run("Merge defaults to root append", func(t *testing.T) {
		p := Merge([]any{1})
		if !p.ShouldMerge() || p.ShouldDeepMerge() {
			t.Fatalf("merge=%v deep=%v", p.ShouldMerge(), p.ShouldDeepMerge())
		}
		got := p.mergeSettings()
		if !got.appendRoot || got.deepMerge {
			t.Fatalf("settings=%+v", got)
		}
	})

	t.Run("DeepMerge constructor", func(t *testing.T) {
		p := DeepMerge(map[string]any{"a": 1})
		if !p.ShouldMerge() || !p.ShouldDeepMerge() {
			t.Fatal("expected deep merge")
		}
		got := p.mergeSettings()
		if !got.deepMerge || got.appendRoot {
			t.Fatalf("settings=%+v", got)
		}
	})

	t.Run("DeepMerge method overrides append", func(t *testing.T) {
		p := Merge("v").Append("data")
		p.DeepMerge()
		got := p.mergeSettings()
		if !got.deepMerge || len(got.appendPaths) != 0 {
			t.Fatalf("settings=%+v", got)
		}
	})

	t.Run("Append and Prepend", func(t *testing.T) {
		p := Merge("v").Append("data").MatchOn("data.id")
		got := p.mergeSettings()
		if !reflect.DeepEqual(got.appendPaths, []string{"data"}) ||
			!reflect.DeepEqual(got.matchesOn, []string{"data.id"}) {
			t.Fatalf("settings=%+v", got)
		}

		p.Prepend("rows")
		got = p.mergeSettings()
		if len(got.appendPaths) != 0 || !reflect.DeepEqual(got.prependPaths, []string{"rows"}) {
			t.Fatalf("after prepend: %+v", got)
		}
		if !reflect.DeepEqual(got.matchesOn, []string{"data.id"}) {
			t.Fatalf("matchesOn should survive: %v", got.matchesOn)
		}
	})

	t.Run("once modifiers", func(t *testing.T) {
		p := Merge("v").Once().As("merge-key").For(time.Second)
		got := p.onceSettings()
		if !got.enabled || got.key != "merge-key" {
			t.Fatalf("settings=%+v", got)
		}
		if got.duration == nil || *got.duration != time.Second {
			t.Fatalf("duration=%v", got.duration)
		}
	})

	t.Run("implements Mergeable", func(t *testing.T) {
		var _ Mergeable = Merge("v")
		var _ Mergeable = DeepMerge("v")
	})
}
