package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestCookieCarryStore_SavePull(t *testing.T) {
	store := NewCookieCarryStore()
	e := echo.New()

	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	if err := store.Save(c, CarryKeyClearHistory, "true"); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(c, CarryKeyPreserveFragment, "true"); err != nil {
		t.Fatal(err)
	}
	cookies := rec.Result().Cookies()
	if len(cookies) == 0 || cookies[len(cookies)-1].Name != CarryCookieName {
		t.Fatalf("cookies=%v", cookies)
	}
	bagCookie := cookies[len(cookies)-1]
	if !bagCookie.HttpOnly || bagCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie attrs=%+v", bagCookie)
	}
	bag, err := decodeCarryBag(bagCookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	if bag[CarryKeyClearHistory] != "true" || bag[CarryKeyPreserveFragment] != "true" {
		t.Fatalf("bag=%v", bag)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: CarryCookieName, Value: bagCookie.Value})
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req, rec2)
	got, ok, err := store.Pull(c2, CarryKeyClearHistory)
	if err != nil || !ok || got != "true" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	got, ok, err = store.Pull(c2, CarryKeyPreserveFragment)
	if err != nil || !ok || got != "true" {
		t.Fatalf("got=%q ok=%v err=%v", got, ok, err)
	}
	cleared := rec2.Result().Cookies()
	if len(cleared) == 0 || cleared[len(cleared)-1].MaxAge != -1 {
		t.Fatalf("clear cookies=%v", cleared)
	}

	c3 := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
	got, ok, err = store.Pull(c3, CarryKeyClearHistory)
	if err != nil || ok || got != "" {
		t.Fatalf("absent pull got=%q ok=%v err=%v", got, ok, err)
	}
}

func testCarryBagCookie(t *testing.T, bag map[string]string) *http.Cookie {
	t.Helper()
	encoded, err := encodeCarryBag(bag)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Cookie{Name: CarryCookieName, Value: encoded}
}
