// Package store provides the notes example's small JSON-backed datastore.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var ErrNotFound = errors.New("record not found")

// A single lock serializes all stores in this process, including file rewrites.
var mutex sync.Mutex

type Store struct {
	dir      string
	users    []User
	notes    []Note
	settings map[string]Settings
}

// OpenWithSeed initializes missing runtime files from seedDir, preserving existing data.
func OpenWithSeed(dir, seedDir string) (*Store, error) {
	mutex.Lock()
	defer mutex.Unlock()
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	for _, name := range []string{"users.json", "notes.json", "settings.json"} {
		target := filepath.Join(dir, name)
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		b, err := os.ReadFile(filepath.Join(seedDir, name))
		if err != nil {
			return nil, fmt.Errorf("read seed %s: %w", name, err)
		}
		if !json.Valid(b) {
			return nil, fmt.Errorf("invalid seed JSON: %s", name)
		}
		if err := initializeFile(dir, target, b); err != nil {
			return nil, fmt.Errorf("initialize %s: %w", name, err)
		}
	}
	return open(dir)
}

// A hard link publishes a complete file atomically and cannot replace a file
// created by another process while this process was reading the seeds.
func initializeFile(dir, target string, b []byte) error {
	f, err := os.CreateTemp(dir, ".seed-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	if _, err := f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Link(f.Name(), target); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return nil
}

func Open(dir string) (*Store, error) {
	mutex.Lock()
	defer mutex.Unlock()
	return open(dir)
}

func open(dir string) (*Store, error) {
	s := &Store{dir: dir, settings: make(map[string]Settings)}
	for _, item := range []struct {
		name  string
		value any
	}{{"users.json", &s.users}, {"notes.json", &s.notes}, {"settings.json", &s.settings}} {
		b, err := os.ReadFile(filepath.Join(dir, item.name))
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", item.name, err)
		}
		if err := json.Unmarshal(b, item.value); err != nil {
			return nil, fmt.Errorf("decode %s: %w", item.name, err)
		}
	}
	if s.settings == nil {
		s.settings = make(map[string]Settings)
	}
	return s, nil
}

func (s *Store) save(name string, value any) error {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(s.dir, "."+name+"-*")
	if err != nil {
		return err
	}
	path := f.Name()
	defer os.Remove(path)
	if _, err = f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	return os.Rename(path, filepath.Join(s.dir, name))
}
