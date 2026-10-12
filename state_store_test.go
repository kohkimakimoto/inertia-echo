package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestCookieStateStore_LoadSave(t *testing.T) {
	store := NewCookieStateStore()
	e := echo.New()

	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	bag := map[string]string{
		StateKeyClearHistory:     "true",
		StateKeyPreserveFragment: "true",
	}
	if err := store.Save(c, bag); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[len(cookies)-1].Name != DefaultCookieStateStoreConfig.Name {
		t.Fatalf("cookies=%v", cookies)
	}
	bagCookie := cookies[len(cookies)-1]
	if !bagCookie.HttpOnly || bagCookie.SameSite != http.SameSiteLaxMode || bagCookie.Path != "/" {
		t.Fatalf("cookie attrs=%+v", bagCookie)
	}
	decoded, err := decodeStateBag(bagCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if decoded[StateKeyClearHistory] != "true" || decoded[StateKeyPreserveFragment] != "true" {
		t.Fatalf("bag=%v", decoded)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: DefaultCookieStateStoreConfig.Name, Value: bagCookie.Value})
	c2 := e.NewContext(req, httptest.NewRecorder())
	loaded, err := store.Load(c2)
	if err != nil {
		t.Fatal(err)
	}
	if loaded[StateKeyClearHistory] != "true" || loaded[StateKeyPreserveFragment] != "true" {
		t.Fatalf("loaded=%v", loaded)
	}

	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec3)
	if err := store.Save(c3, map[string]string{}); err != nil {
		t.Fatal(err)
	}
	cleared := rec3.Result().Cookies()
	if len(cleared) == 0 || cleared[len(cleared)-1].MaxAge != -1 {
		t.Fatalf("clear cookies=%v", cleared)
	}

	c4 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	empty, err := store.Load(c4)
	if err != nil || len(empty) != 0 {
		t.Fatalf("empty load=%v err=%v", empty, err)
	}
}

func TestCookieStateStore_WithConfig(t *testing.T) {
	cfg := DefaultCookieStateStoreConfig
	cfg.Name = "custom.state"
	cfg.Path = "/app"
	cfg.Domain = "example.com"
	cfg.Secure = true
	cfg.SameSite = http.SameSiteStrictMode
	store := NewCookieStateStoreWithConfig(cfg)

	rec := httptest.NewRecorder()
	c := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	if err := store.Save(c, map[string]string{StateKeyClearHistory: "true"}); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected cookie")
	}
	got := cookies[len(cookies)-1]
	if got.Name != "custom.state" || got.Path != "/app" || got.Domain != "example.com" ||
		!got.Secure || !got.HttpOnly || got.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie attrs=%+v", got)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: "custom.state", Value: got.Value})
	loaded, err := store.Load(echo.New().NewContext(req, httptest.NewRecorder()))
	if err != nil || loaded[StateKeyClearHistory] != "true" {
		t.Fatalf("loaded=%v err=%v", loaded, err)
	}

	recClear := httptest.NewRecorder()
	if err := store.Save(echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), recClear), map[string]string{}); err != nil {
		t.Fatal(err)
	}
	cleared := recClear.Result().Cookies()[0]
	if cleared.MaxAge != -1 || cleared.Path != "/app" || cleared.Domain != "example.com" || !cleared.Secure {
		t.Fatalf("clear cookie attrs=%+v", cleared)
	}
}

func TestInertiaStateBagOps(t *testing.T) {
	i := &Inertia{state: map[string]string{}}

	i.stateSet(StateKeyClearHistory, "true")
	if !i.stateDirty {
		t.Fatal("expected dirty after Set")
	}
	got, ok := i.stateGet(StateKeyClearHistory)
	if !ok || got != "true" {
		t.Fatalf("Get got=%q ok=%v", got, ok)
	}

	pulled, ok := i.statePull(StateKeyClearHistory)
	if !ok || pulled != "true" {
		t.Fatalf("Pull got=%q ok=%v", pulled, ok)
	}
	if _, ok := i.stateGet(StateKeyClearHistory); ok {
		t.Fatal("expected missing after Pull")
	}

	i.stateSet(StateKeyPreserveFragment, "true")
	i.stateDelete(StateKeyPreserveFragment)
	if _, ok := i.stateGet(StateKeyPreserveFragment); ok {
		t.Fatal("expected missing after Delete")
	}
}

func testStateBagCookie(t *testing.T, bag map[string]string) *http.Cookie {
	t.Helper()
	encoded, err := encodeStateBag(bag)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: DefaultCookieStateStoreConfig.Name, Value: encoded}
}
