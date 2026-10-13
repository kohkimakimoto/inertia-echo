package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"users.json", "notes.json", "settings.json"} {
		b, err := os.ReadFile(filepath.Join("..", "..", "seed", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestNotesPersistenceAndIsolation(t *testing.T) {
	s := testStore(t)
	n, err := s.CreateNote("u1", NoteInput{Title: "Persist me", Body: "Searchable text", Tags: []string{" GO ", "go", "", "Demo"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(n.Tags) != 2 || n.Tags[0] != "go" {
		t.Fatalf("tags: %v", n.Tags)
	}
	n.Tags[0] = "mutated"
	got, _ := s.GetNote("u1", n.ID)
	if got.Tags[0] != "go" {
		t.Fatal("caller mutated stored tags")
	}
	if _, ok := s.GetNote("u2", n.ID); ok {
		t.Fatal("cross-user read")
	}
	if _, err := s.UpdateNote("u2", n.ID, NoteInput{}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user update: %v", err)
	}
	if _, err := s.TogglePin("u2", n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user pin: %v", err)
	}
	if err := s.DeleteNote("u2", n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-user delete: %v", err)
	}
	got, err = s.UpdateNote("u1", n.ID, NoteInput{Title: "Changed", Tags: []string{"updated"}})
	if err != nil {
		t.Fatal(err)
	}
	if !got.CreatedAt.Equal(n.CreatedAt) {
		t.Fatal("creation timestamp changed")
	}
	if _, err = s.TogglePin("u1", n.ID); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := reopened.GetNote("u1", n.ID)
	if !ok || got.Title != "Changed" || !got.Pinned {
		t.Fatalf("persisted note: %+v", got)
	}
	if err := reopened.DeleteNote("u1", n.ID); err != nil {
		t.Fatal(err)
	}
	reopened, err = Open(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.GetNote("u1", n.ID); ok {
		t.Fatal("delete not persisted")
	}
}
func TestFiltersAndOrdering(t *testing.T) {
	s := testStore(t)
	notes, total := s.ListNotes("u1", Filter{Limit: 10})
	if total != 32 || len(notes) != 10 {
		t.Fatalf("pagination: %d/%d", len(notes), total)
	}
	if !notes[0].Pinned || !notes[1].Pinned || notes[2].Pinned {
		t.Fatal("pinned ordering")
	}
	notes, total = s.ListNotes("u1", Filter{Query: "SEASONAL", Tag: "PERSONAL"})
	if total == 0 {
		t.Fatal("case-insensitive filter returned nothing")
	}
	for _, n := range notes {
		if n.UserID != "u1" {
			t.Fatal("cross-user listing")
		}
	}
	notes, total = s.ListNotes("u1", Filter{Offset: 1000, Limit: 10})
	if total != 32 || len(notes) != 0 {
		t.Fatal("offset beyond end")
	}
}
func TestSettingsPersistAndValidate(t *testing.T) {
	s := testStore(t)
	want := Settings{Theme: "dark", PageSize: 25}
	if err := s.SaveSettings("u1", want); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(s.dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := reopened.GetSettings("u1"); got != want {
		t.Fatalf("settings: %+v", got)
	}
	if got := reopened.GetSettings("u2"); got.Theme != "light" || got.PageSize != 10 {
		t.Fatalf("other user's settings changed: %+v", got)
	}
	if err := s.SaveSettings("u1", Settings{Theme: "invalid", PageSize: 10}); err == nil {
		t.Fatal("invalid theme accepted")
	}
	if err := s.SaveSettings("u1", Settings{Theme: "dark", PageSize: 0}); err == nil {
		t.Fatal("invalid page size accepted")
	}
	if err := s.SaveSettings("missing", want); !errors.Is(err, ErrNotFound) {
		t.Fatal("missing user accepted")
	}
}
func TestFailedWriteKeepsMemory(t *testing.T) {
	s := testStore(t)
	before, total := s.ListNotes("u1", Filter{})
	s.dir = filepath.Join(s.dir, "missing")
	if _, err := s.CreateNote("u1", NoteInput{Title: "Do not retain"}); err == nil {
		t.Fatal("expected write error")
	}
	after, afterTotal := s.ListNotes("u1", Filter{})
	if total != afterTotal || len(before) != len(after) {
		t.Fatal("failed write mutated memory")
	}
}

func TestInitializeRuntimeFromSeedAndPreserveChanges(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "runtime")
	seedDir := filepath.Join("..", "..", "seed")
	s, err := OpenWithSeed(dir, seedDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, count := s.ListNotes("u1", Filter{}); count != 32 {
		t.Fatalf("seed note count: %d", count)
	}
	n, err := s.CreateNote("u1", NoteInput{Title: "Keep runtime changes"})
	if err != nil {
		t.Fatal(err)
	}
	want := Settings{Theme: "dark", PageSize: 25}
	if err := s.SaveSettings("u1", want); err != nil {
		t.Fatal(err)
	}
	// Existing runtime files remain usable even when the seed directory is absent.
	reopened, err := OpenWithSeed(dir, filepath.Join(t.TempDir(), "missing-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.GetNote("u1", n.ID); !ok {
		t.Fatal("initialization overwrote notes")
	}
	if got := reopened.GetSettings("u1"); got != want {
		t.Fatalf("initialization overwrote settings: %+v", got)
	}
	// Repair only a missing file without disturbing the other runtime files.
	if err := os.Remove(filepath.Join(dir, "users.json")); err != nil {
		t.Fatal(err)
	}
	repaired, err := OpenWithSeed(dir, seedDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := repaired.GetUser("u1"); !ok {
		t.Fatal("missing users file not initialized")
	}
	if _, ok := repaired.GetNote("u1", n.ID); !ok {
		t.Fatal("partial initialization overwrote notes")
	}
	if got := repaired.GetSettings("u1"); got != want {
		t.Fatal("partial initialization overwrote settings")
	}
}
