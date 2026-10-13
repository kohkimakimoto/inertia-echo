package store

import "strings"

type User struct {
	ID       string `json:"id"`
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

func (s *Store) FindUserByEmail(email string) (User, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, u := range s.users {
		if strings.EqualFold(u.Email, strings.TrimSpace(email)) {
			return u, true
		}
	}
	return User{}, false
}
func (s *Store) GetUser(id string) (User, bool) {
	mutex.Lock()
	defer mutex.Unlock()
	for _, u := range s.users {
		if u.ID == id {
			return u, true
		}
	}
	return User{}, false
}
