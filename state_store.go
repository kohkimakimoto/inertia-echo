package inertia

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/labstack/echo/v5"
)

// StateStore logical keys inside the in-memory bag on Inertia (not cookie names).
// Unlike inertia-laravel's SessionKey values (which use an "inertia." prefix because
// they share Laravel's app session), this bag is Inertia-owned, so bare names suffice.
// They are not configurable — Laravel fixes its SessionKey names too.
const (
	StateKeyClearHistory     = "clear_history"
	StateKeyFlashData        = "flash_data"
	StateKeyPreserveFragment = "preserve_fragment"
)

// StateStore persists Inertia's request-scoped state bag across HTTP requests.
//
// Design intent:
// Echo has no built-in session, but several Inertia core features assume one
// (flash data, clear-history, preserve-fragment — inertia-laravel stores these
// in the Laravel session). StateStore fills only that gap: an Inertia-owned
// bag that survives until a later request. Get/Set/Delete/Pull run in memory
// on the request-scoped Inertia; StateStore only Load/Save the bag.
// It is intentionally not a general-purpose session, so it does not collide
// with or double-manage whatever session system an application already uses.
//
// Security: this is not a secure session. The default cookie implementation is
// unsigned and client-controlled. Values loaded from StateStore must be treated
// like any other untrusted request input (query, form, header). Do not use them
// for authorization, user identity, or secrets. Suitable for low-risk state such
// as flash display data and boolean UI flags.
type StateStore interface {
	// Load returns the bag for this request. Missing or broken storage yields
	// an empty bag (non-nil map), not an error.
	Load(c *echo.Context) (map[string]string, error)
	// Save persists bag. An empty bag clears stored state.
	Save(c *echo.Context, bag map[string]string) error
}

// CookieStateStoreConfig configures cookie attributes for CookieStateStore.
// Zero Name, Path, or SameSite are replaced with DefaultCookieStateStoreConfig
// values. Other fields are used as provided — copy the default before overriding
// a subset of fields so HTTPOnly stays true:
//
//	cfg := inertia.DefaultCookieStateStoreConfig
//	cfg.Secure = true
//	store := inertia.NewCookieStateStoreWithConfig(cfg)
type CookieStateStoreConfig struct {
	Name     string
	Path     string
	Domain   string
	Secure   bool
	HTTPOnly bool
	SameSite http.SameSite
}

// DefaultCookieStateStoreConfig is used by NewCookieStateStore.
var DefaultCookieStateStoreConfig = CookieStateStoreConfig{
	Name:     "inertia.state",
	Path:     "/",
	HTTPOnly: true,
	SameSite: http.SameSiteLaxMode,
}

// CookieStateStore is the default StateStore. It keeps the bag in a single
// unsigned cookie, like a tiny session bag.
//
// Security: cookie values are not authenticated. A client can forge or alter
// them. Treat Load results as untrusted input, same as request parameters.
// Do not store credentials, session tokens, or authorization state here.
type CookieStateStore struct {
	config CookieStateStoreConfig
}

// NewCookieStateStore returns the default unsigned cookie StateStore.
func NewCookieStateStore() *CookieStateStore {
	return NewCookieStateStoreWithConfig(DefaultCookieStateStoreConfig)
}

// NewCookieStateStoreWithConfig returns a CookieStateStore with the given
// cookie attributes. See CookieStateStoreConfig for zero-value handling.
func NewCookieStateStoreWithConfig(config CookieStateStoreConfig) *CookieStateStore {
	if config.Name == "" {
		config.Name = DefaultCookieStateStoreConfig.Name
	}
	if config.Path == "" {
		config.Path = DefaultCookieStateStoreConfig.Path
	}
	if config.SameSite == 0 {
		config.SameSite = DefaultCookieStateStoreConfig.SameSite
	}
	return &CookieStateStore{config: config}
}

func (s *CookieStateStore) Load(c *echo.Context) (map[string]string, error) {
	bag := map[string]string{}
	cookie, err := c.Request().Cookie(s.config.Name)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			return nil, err
		}
		return bag, nil
	}
	if cookie.Value == "" {
		return bag, nil
	}
	decoded, err := decodeStateBag(cookie.Value)
	if err != nil {
		// Broken payloads are ignored; Save will replace or clear.
		return bag, nil
	}
	return decoded, nil
}

func (s *CookieStateStore) Save(c *echo.Context, bag map[string]string) error {
	if len(bag) == 0 {
		replaceCookie(c.Response(), s.cookie("", -1))
		return nil
	}
	encoded, err := encodeStateBag(bag)
	if err != nil {
		return err
	}
	replaceCookie(c.Response(), s.cookie(encoded, 0))
	return nil
}

func (s *CookieStateStore) cookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     s.config.Name,
		Value:    value,
		Path:     s.config.Path,
		Domain:   s.config.Domain,
		Secure:   s.config.Secure,
		HttpOnly: s.config.HTTPOnly,
		SameSite: s.config.SameSite,
		MaxAge:   maxAge,
	}
}

// replaceCookie sets cookie, replacing any prior Set-Cookie for the same name
// so multiple Save calls in one request do not stack stale bag snapshots.
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

func encodeStateBag(bag map[string]string) (string, error) {
	b, err := json.Marshal(bag)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func decodeStateBag(raw string) (map[string]string, error) {
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
