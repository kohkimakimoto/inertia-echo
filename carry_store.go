package inertia

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// CarryStore keys mirror inertia-laravel's Inertia\Support\SessionKey constants.
// They are logical keys inside the carry bag (not cookie names). Laravel stores
// them in the session bag; CookieCarryStore stores them in one cookie payload.
// They are not configurable — Laravel fixes them too.
const (
	CarryKeyClearHistory     = "inertia.clear_history"
	CarryKeyFlashData        = "inertia.flash_data"
	CarryKeyPreserveFragment = "inertia.preserve_fragment"
)

// CarryCookieName is the single cookie that holds the CarryStore bag for the
// default CookieCarryStore. Logical keys (CarryKey*) live inside its payload,
// analogous to keys inside Laravel's session.
const CarryCookieName = "inertia.carry"

const carryBagContextKey = "__inertia_carry_bag__"

// CarryStore persists one-shot string values across redirects until pulled.
//
// Design intent:
// Echo has no built-in session, but several Inertia core features assume one
// (flash data, clear-history, preserve-fragment — inertia-laravel stores these
// in the Laravel session). CarryStore fills only that gap: an Inertia-owned
// carry bag for values that must survive a redirect until the next Page render.
// It is intentionally not a general-purpose session, so it does not collide
// with or double-manage whatever session system an application already uses.
//
// Security: this is not a secure session. The default cookie implementation is
// unsigned and client-controlled. Values returned by Pull must be treated like
// any other untrusted request input (query, form, header). Do not use them for
// authorization, user identity, or secrets. Suitable for low-risk carry such as
// flash display data and boolean UI flags.
type CarryStore interface {
	// Pull reads and removes the value for key. ok is false when absent.
	Pull(c *echo.Context, key string) (value string, ok bool, err error)
	// Save writes value for key so a later request can Pull it.
	Save(c *echo.Context, key string, value string) error
}

// CookieCarryStore is the default CarryStore. It keeps all logical keys in a
// single unsigned HttpOnly cookie (CarryCookieName), like a tiny session bag.
//
// Security: cookie values are not authenticated. A client can forge or alter
// them. Treat Pull results as untrusted input, same as request parameters.
// Do not store credentials, session tokens, or authorization state here.
type CookieCarryStore struct{}

// NewCookieCarryStore returns the default unsigned cookie CarryStore.
func NewCookieCarryStore() *CookieCarryStore {
	return &CookieCarryStore{}
}

func (s *CookieCarryStore) Pull(c *echo.Context, key string) (string, bool, error) {
	bag, err := s.bag(c)
	if err != nil {
		return "", false, err
	}
	value, ok := bag[key]
	if !ok {
		return "", false, nil
	}
	delete(bag, key)
	if err := s.writeBag(c, bag); err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *CookieCarryStore) Save(c *echo.Context, key string, value string) error {
	bag, err := s.bag(c)
	if err != nil {
		return err
	}
	bag[key] = value
	return s.writeBag(c, bag)
}

func (s *CookieCarryStore) bag(c *echo.Context) (map[string]string, error) {
	if v := c.Get(carryBagContextKey); v != nil {
		bag, ok := v.(map[string]string)
		if ok {
			return bag, nil
		}
	}
	bag := map[string]string{}
	cookie, err := c.Request().Cookie(CarryCookieName)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			return nil, err
		}
	} else if cookie.Value != "" {
		decoded, err := decodeCarryBag(cookie.Value)
		if err == nil {
			bag = decoded
		}
		// Broken payloads are ignored; writeBag/Pull will replace or clear.
	}
	c.Set(carryBagContextKey, bag)
	return bag, nil
}

func (s *CookieCarryStore) writeBag(c *echo.Context, bag map[string]string) error {
	if len(bag) == 0 {
		replaceCookie(c.Response(), &http.Cookie{
			Name:     CarryCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})
		return nil
	}
	encoded, err := encodeCarryBag(bag)
	if err != nil {
		return err
	}
	replaceCookie(c.Response(), &http.Cookie{
		Name:     CarryCookieName,
		Value:    encoded,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
	return nil
}

// replaceCookie sets cookie, replacing any prior Set-Cookie for the same name
// so multiple Save/Pull calls in one request do not stack stale bag snapshots.
func replaceCookie(w http.ResponseWriter, cookie *http.Cookie) {
	h := w.Header()
	existing := h.Values("Set-Cookie")
	if len(existing) > 0 {
		h.Del("Set-Cookie")
		prefix := cookie.Name + "="
		for _, v := range existing {
			if !strings.HasPrefix(v, prefix) {
				h.Add("Set-Cookie", v)
			}
		}
	}
	http.SetCookie(w, cookie)
}

func encodeCarryBag(bag map[string]string) (string, error) {
	b, err := json.Marshal(bag)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeCarryBag(raw string) (map[string]string, error) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	var bag map[string]string
	if err := json.Unmarshal(b, &bag); err != nil {
		return nil, err
	}
	if bag == nil {
		bag = map[string]string{}
	}
	return bag, nil
}
