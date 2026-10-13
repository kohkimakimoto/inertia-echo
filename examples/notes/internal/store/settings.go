package store

import "errors"

type Settings struct {
	Theme    string `json:"theme"`
	PageSize int    `json:"page_size"`
}

func (s *Store) GetSettings(userID string) Settings {
	mutex.Lock()
	defer mutex.Unlock()
	if v, ok := s.settings[userID]; ok {
		return v
	}
	return Settings{Theme: "system", PageSize: 10}
}
func (s *Store) SaveSettings(userID string, v Settings) error {
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
		return ErrNotFound
	}
	if v.Theme != "system" && v.Theme != "light" && v.Theme != "dark" {
		return errors.New("invalid theme")
	}
	if v.PageSize < 5 || v.PageSize > 50 {
		return errors.New("page size must be between 5 and 50")
	}
	next := make(map[string]Settings, len(s.settings)+1)
	for k, val := range s.settings {
		next[k] = val
	}
	next[userID] = v
	if err := s.save("settings.json", next); err != nil {
		return err
	}
	s.settings = next
	return nil
}
