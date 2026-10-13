package store

import (
	"crypto/rand"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type Note struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags"`
	Pinned    bool      `json:"pinned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
type NoteInput struct {
	Title  string   `json:"title" form:"title"`
	Body   string   `json:"body" form:"body"`
	Tags   []string `json:"tags" form:"tags"`
	Pinned bool     `json:"pinned" form:"pinned"`
}
type Filter struct {
	Query  string
	Tag    string
	Offset int
	Limit  int
}

func clone(n Note) Note { n.Tags = append([]string{}, n.Tags...); return n }
func normalizeTags(tags []string) []string {
	result := []string{}
	seen := map[string]bool{}
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag != "" && !seen[tag] {
			seen[tag] = true
			result = append(result, tag)
		}
	}
	return result
}
func (s *Store) ListNotes(userID string, f Filter) ([]Note, int) {
	mutex.Lock()
	defer mutex.Unlock()
	result := []Note{}
	query, tag := strings.ToLower(strings.TrimSpace(f.Query)), strings.ToLower(strings.TrimSpace(f.Tag))
	for _, n := range s.notes {
		if n.UserID != userID {
			continue
		}
		if query != "" && !strings.Contains(strings.ToLower(n.Title+"\n"+n.Body), query) {
			continue
		}
		if tag != "" {
			found := false
			for _, t := range n.Tags {
				if t == tag {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, clone(n))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Pinned != result[j].Pinned {
			return result[i].Pinned
		}
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	total := len(result)
	start := max(0, min(f.Offset, total))
	end := total
	if f.Limit > 0 {
		end = start + min(f.Limit, total-start)
	}
	return result[start:end], total
}
func (s *Store) Tags(userID string) []string {
	notes, _ := s.ListNotes(userID, Filter{})
	set := map[string]bool{}
	result := []string{}
	for _, n := range notes {
		for _, t := range n.Tags {
			if !set[t] {
				set[t] = true
				result = append(result, t)
			}
		}
	}
	sort.Strings(result)
	return result
}
func (s *Store) GetNote(userID, id string) (Note, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, n := range s.notes {
		if n.UserID == userID && n.ID == id {
			return clone(n), true
		}
	}
	return Note{}, false
}
func (s *Store) CreateNote(userID string, in NoteInput) (Note, error) {
	mutex.Lock()
	defer mutex.Unlock()
	found := false
	for _, u := range s.users {
		if u.ID == userID {
			found = true
			break
		}
	}
	if !found {
		return Note{}, ErrNotFound
	}
	var bytes [12]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return Note{}, err
	}
	now := time.Now().UTC()
	n := Note{ID: "n" + hex.EncodeToString(bytes[:]), UserID: userID, Title: in.Title, Body: in.Body, Tags: normalizeTags(in.Tags), Pinned: in.Pinned, CreatedAt: now, UpdatedAt: now}
	next := append(append([]Note{}, s.notes...), n)
	if err := s.save("notes.json", next); err != nil {
		return Note{}, err
	}
	s.notes = next
	return clone(n), nil
}
func (s *Store) UpdateNote(userID, id string, in NoteInput) (Note, error) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, n := range s.notes {
		if n.UserID == userID && n.ID == id {
			n.Title = in.Title
			n.Body = in.Body
			n.Tags = normalizeTags(in.Tags)
			n.Pinned = in.Pinned
			n.UpdatedAt = time.Now().UTC()
			next := append([]Note{}, s.notes...)
			next[i] = n
			if err := s.save("notes.json", next); err != nil {
				return Note{}, err
			}
			s.notes = next
			return clone(n), nil
		}
	}
	return Note{}, ErrNotFound
}
func (s *Store) TogglePin(userID, id string) (Note, error) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, n := range s.notes {
		if n.UserID == userID && n.ID == id {
			n.Pinned = !n.Pinned
			n.UpdatedAt = time.Now().UTC()
			next := append([]Note{}, s.notes...)
			next[i] = n
			if err := s.save("notes.json", next); err != nil {
				return Note{}, err
			}
			s.notes = next
			return clone(n), nil
		}
	}
	return Note{}, ErrNotFound
}
func (s *Store) DeleteNote(userID, id string) error {
	mutex.Lock()
	defer mutex.Unlock()
	for i, n := range s.notes {
		if n.UserID == userID && n.ID == id {
			next := append([]Note{}, s.notes[:i]...)
			next = append(next, s.notes[i+1:]...)
			if err := s.save("notes.json", next); err != nil {
				return err
			}
			s.notes = next
			return nil
		}
	}
	return ErrNotFound
}
