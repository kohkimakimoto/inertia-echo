package inertia

import (
	"testing"
	"time"
)

func TestOnceSettingsExpiresAt(t *testing.T) {
	now := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.UTC)

	t.Run("neither until nor for yields nil", func(t *testing.T) {
		s := onceSettings{enabled: true}
		if got := s.expiresAt(now); got != nil {
			t.Fatalf("expiresAt=%v, want nil", got)
		}
	})

	t.Run("until uses absolute timestamp", func(t *testing.T) {
		until := now.Add(5 * time.Minute)
		s := onceSettings{}
		s.setUntil(until)
		got := s.expiresAt(now)
		if got == nil {
			t.Fatal("expected expiresAt")
		}
		if *got != until.UnixMilli() {
			t.Fatalf("expiresAt=%d, want %d", *got, until.UnixMilli())
		}
	})

	t.Run("for uses relative duration from now", func(t *testing.T) {
		s := onceSettings{}
		s.setFor(90 * time.Second)
		got := s.expiresAt(now)
		if got == nil {
			t.Fatal("expected expiresAt")
		}
		want := now.Add(90 * time.Second).UnixMilli()
		if *got != want {
			t.Fatalf("expiresAt=%d, want %d", *got, want)
		}
	})

	t.Run("for clears until", func(t *testing.T) {
		s := onceSettings{}
		s.setUntil(now.Add(time.Hour))
		s.setFor(time.Minute)
		if s.until != nil {
			t.Fatal("until should be cleared by setFor")
		}
		if s.duration == nil || *s.duration != time.Minute {
			t.Fatalf("duration=%v", s.duration)
		}
	})

	t.Run("until clears for", func(t *testing.T) {
		s := onceSettings{}
		s.setFor(time.Minute)
		until := now.Add(time.Hour)
		s.setUntil(until)
		if s.duration != nil {
			t.Fatal("duration should be cleared by setUntil")
		}
		if s.until == nil || !s.until.Equal(until) {
			t.Fatalf("until=%v", s.until)
		}
	})
}

func TestOnceSettingsModifiersEnable(t *testing.T) {
	t.Run("setKey", func(t *testing.T) {
		s := onceSettings{}
		s.setKey("custom")
		if !s.enabled || s.key != "custom" {
			t.Fatalf("settings=%+v", s)
		}
	})

	t.Run("setFresh", func(t *testing.T) {
		s := onceSettings{}
		s.setFresh(true)
		if !s.enabled || !s.fresh {
			t.Fatalf("settings=%+v", s)
		}
	})
}

func TestOnceBuilder(t *testing.T) {
	until := time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC)
	prop := Once(func() (any, error) { return "v", nil }).
		As("cache-key").
		Fresh(true).
		Until(until)

	got := prop.onceSettings()
	if !got.enabled || got.key != "cache-key" || !got.fresh {
		t.Fatalf("settings=%+v", got)
	}
	if got.until == nil || !got.until.Equal(until) {
		t.Fatalf("until=%v", got.until)
	}

	prop.For(2 * time.Minute)
	got = prop.onceSettings()
	if got.until != nil {
		t.Fatal("Until should be cleared by For")
	}
	if got.duration == nil || *got.duration != 2*time.Minute {
		t.Fatalf("duration=%v", got.duration)
	}
}
