package inertia

import "testing"

func TestScrollBuilder(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		p := Scroll(func() (ScrollResult, error) {
			return ScrollResult{}, nil
		})
		if p.dataPath != "data" {
			t.Fatalf("dataPath=%q", p.dataPath)
		}
		if p.shouldDefer() {
			t.Fatal("defer should be off by default")
		}
		if p.group() != "default" {
			t.Fatalf("group=%q", p.group())
		}
	})

	t.Run("WithDataPath", func(t *testing.T) {
		p := Scroll(func() (ScrollResult, error) {
			return ScrollResult{}, nil
		}).WithDataPath("items")
		if p.dataPath != "items" {
			t.Fatalf("dataPath=%q", p.dataPath)
		}
	})

	t.Run("Defer enables default group", func(t *testing.T) {
		p := Scroll(func() (ScrollResult, error) {
			return ScrollResult{}, nil
		}).Defer()
		if !p.shouldDefer() || p.group() != "default" {
			t.Fatalf("defer=%v group=%q", p.shouldDefer(), p.group())
		}
	})

	t.Run("Defer with custom group", func(t *testing.T) {
		p := Scroll(func() (ScrollResult, error) {
			return ScrollResult{}, nil
		}).Defer("feed")
		if !p.shouldDefer() || p.group() != "feed" {
			t.Fatalf("defer=%v group=%q", p.shouldDefer(), p.group())
		}
	})

	t.Run("Defer ignores empty group override", func(t *testing.T) {
		p := Scroll(func() (ScrollResult, error) {
			return ScrollResult{}, nil
		}).Defer("")
		if !p.shouldDefer() || p.group() != "default" {
			t.Fatalf("defer=%v group=%q", p.shouldDefer(), p.group())
		}
	})
}
