package inertia

import (
	"reflect"
	"testing"
)

func TestMergeSettingsSetAppendPrepend(t *testing.T) {
	t.Run("append root", func(t *testing.T) {
		s := mergeSettings{matchesOn: []string{"id"}, deepMerge: true}
		s.setAppend(nil)
		if !s.appendRoot || s.deepMerge || len(s.appendPaths) != 0 || len(s.prependPaths) != 0 {
			t.Fatalf("settings=%+v", s)
		}
		if !reflect.DeepEqual(s.matchesOn, []string{"id"}) {
			t.Fatalf("matchesOn should be preserved: %v", s.matchesOn)
		}
	})

	t.Run("append paths", func(t *testing.T) {
		s := mergeSettings{}
		s.setAppend([]string{"data", "items"})
		if s.appendRoot || !reflect.DeepEqual(s.appendPaths, []string{"data", "items"}) {
			t.Fatalf("settings=%+v", s)
		}
	})

	t.Run("prepend root", func(t *testing.T) {
		s := mergeSettings{matchesOn: []string{"id"}}
		s.setPrepend(nil)
		if !s.prependRoot || len(s.prependPaths) != 0 {
			t.Fatalf("settings=%+v", s)
		}
		if !reflect.DeepEqual(s.matchesOn, []string{"id"}) {
			t.Fatalf("matchesOn should be preserved: %v", s.matchesOn)
		}
	})

	t.Run("prepend paths replaces append", func(t *testing.T) {
		s := mergeSettings{}
		s.setAppend([]string{"data"})
		s.setPrepend([]string{"rows"})
		if s.appendRoot || len(s.appendPaths) != 0 {
			t.Fatalf("append should be cleared: %+v", s)
		}
		if !reflect.DeepEqual(s.prependPaths, []string{"rows"}) {
			t.Fatalf("settings=%+v", s)
		}
	})
}

func TestMergeSettingsEnabled(t *testing.T) {
	if rootAppendMergeSettings().enabled() != true {
		t.Fatal("root append should be enabled")
	}
	if (mergeSettings{}).enabled() {
		t.Fatal("empty settings should be disabled")
	}
	if !(mergeSettings{prependPaths: []string{"x"}}).enabled() {
		t.Fatal("prepend paths should enable")
	}
}
