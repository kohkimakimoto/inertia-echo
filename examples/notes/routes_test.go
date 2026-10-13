package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	session "github.com/kohkimakimoto/echo-session/v5"
	"github.com/kohkimakimoto/inertia-echo/examples/notes/internal/store"
	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/labstack/echo/v5"
)

type testClient struct {
	t       *testing.T
	e       *echo.Echo
	cookies map[string]*http.Cookie
	db      *store.Store
}

func newClient(t *testing.T) *testClient {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"users.json", "notes.json", "settings.json"} {
		b, err := os.ReadFile(filepath.Join("seed", name))
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, name), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	db, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()

	e.Use(session.Middleware(session.NewCookieStore([]byte("test-secret"))))
	e.Use(inertia.MiddlewareWithConfig(inertia.MiddlewareConfig{Renderer: inertia.NewHTMLRenderer(), VersionFunc: func(*echo.Context) string { return "" }}))
	e.Use(inertia.CSRF())
	e.Use(inertia.EncryptHistoryMiddleware())
	registerRoutes(e, db)
	return &testClient{t: t, e: e, cookies: map[string]*http.Cookie{}, db: db}
}
func (c *testClient) request(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	c.t.Helper()
	var b bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&b).Encode(body); err != nil {
			c.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, path, &b)
	req.Header.Set("X-Inertia", "true")
	req.Header.Set("Content-Type", "application/json")
	for _, cookie := range c.cookies {
		req.AddCookie(cookie)
	}
	if cookie := c.cookies["XSRF-TOKEN"]; cookie != nil {
		req.Header.Set("X-XSRF-TOKEN", cookie.Value)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	c.e.ServeHTTP(rec, req)
	for _, cookie := range rec.Result().Cookies() {
		if cookie.MaxAge < 0 || (!cookie.Expires.IsZero() && cookie.Expires.Before(time.Now())) {
			delete(c.cookies, cookie.Name)
		} else {
			c.cookies[cookie.Name] = cookie
		}
	}
	return rec
}
func page(t *testing.T, r *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var p map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &p); err != nil {
		t.Fatalf("status %d: %s", r.Code, r.Body.String())
	}
	return p
}
func props(t *testing.T, r *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	return page(t, r)["props"].(map[string]any)
}
func (c *testClient) login(email string) {
	c.t.Helper()
	c.request("GET", "/login", nil, nil)
	r := c.request("POST", "/login", map[string]any{"email": email, "password": "password"}, nil)
	if r.Code != 303 || r.Header().Get("Location") != "/" {
		c.t.Fatalf("login: %d %s", r.Code, r.Body.String())
	}
	found := false
	for _, cookie := range r.Result().Cookies() {
		if cookie.Name == "session" {
			found = true
			if cookie.MaxAge <= 0 || cookie.Value == "" || (!cookie.Expires.IsZero() && !cookie.Expires.After(time.Now())) {
				c.t.Fatalf("login issued an expired session cookie: %+v", cookie)
			}
		}
	}
	if !found || c.cookies["session"] == nil {
		c.t.Fatal("login did not retain a session cookie")
	}
}
func TestAuthCSRFAndLogout(t *testing.T) {
	c := newClient(t)
	if r := c.request("GET", "/notes", nil, nil); r.Code != 303 || r.Header().Get("Location") != "/login" {
		t.Fatalf("guest access: %d %s", r.Code, r.Body.String())
	}
	if r := c.request("POST", "/login", map[string]string{"email": "demo@example.com", "password": "password"}, map[string]string{"X-XSRF-TOKEN": ""}); r.Code != 400 && r.Code != 403 {
		t.Fatalf("missing CSRF accepted: %d", r.Code)
	}
	c.request("GET", "/login", nil, nil)
	r := c.request("POST", "/login", map[string]string{"email": "demo@example.com", "password": "wrong"}, nil)
	if props(t, r)["errors"].(map[string]any)["email"] == nil {
		t.Fatal("missing login validation")
	}
	c.login("demo@example.com")
	p := page(t, c.request("GET", "/", nil, nil))
	if p["clearHistory"] != true || p["encryptHistory"] != true {
		t.Fatalf("history flags: %v", p)
	}
	if r := c.request("POST", "/logout", nil, nil); r.Code != 303 {
		t.Fatalf("logout: %d", r.Code)
	}
	if c.cookies["session"] != nil {
		t.Fatal("logout did not delete the session cookie")
	}
	if r := c.request("GET", "/notes", nil, nil); r.Header().Get("Location") != "/login" {
		t.Fatal("logout did not clear session")
	}
}
func TestNoteCRUDValidationAndOwnership(t *testing.T) {
	c := newClient(t)
	c.login("demo@example.com")
	r := c.request("POST", "/notes", map[string]any{"title": " ", "body": "Keep this draft", "tags": []string{}}, nil)
	p := props(t, r)
	if p["errors"].(map[string]any)["title"] == nil || p["note"].(map[string]any)["body"] != "Keep this draft" {
		t.Fatal("validation lost submitted draft")
	}
	r = c.request("POST", "/notes", map[string]any{"title": "HTTP note", "body": "Body", "tags": []string{"GO", "go"}, "pinned": true}, nil)
	if r.Code != 303 {
		t.Fatalf("create: %d %s", r.Code, r.Body.String())
	}
	path := r.Header().Get("Location")
	id := strings.TrimPrefix(path, "/notes/")
	n, ok := c.db.GetNote("u1", id)
	if !ok || n.Title != "HTTP note" || len(n.Tags) != 1 {
		t.Fatalf("created note: %+v", n)
	}
	r = c.request("POST", path, map[string]any{"title": "Updated", "body": "New body", "tags": []string{}}, nil)
	if r.Code != 303 {
		t.Fatalf("update: %d", r.Code)
	}
	for _, path := range []string{"/notes/alice-1", "/notes/alice-1/edit"} {
		r = c.request("GET", path, nil, nil)
		if r.Code != 404 || page(t, r)["component"] != "Errors/NotFound" {
			t.Fatalf("ownership read: %d", r.Code)
		}
	}
	for _, suffix := range []string{"", "/pin", "/delete", "/fragment"} {
		r = c.request("POST", "/notes/alice-1"+suffix, map[string]any{"title": "stolen"}, nil)
		if r.Code != 404 {
			t.Fatalf("ownership mutation %s: %d", suffix, r.Code)
		}
	}
	r = c.request("POST", path+"/delete", nil, nil)
	if r.Code != 303 {
		t.Fatalf("delete: %d", r.Code)
	}
	if _, ok := c.db.GetNote("u1", id); ok {
		t.Fatal("delete did not persist")
	}
	if r = c.request("GET", "/notes/missing", nil, nil); r.Code != 404 {
		t.Fatal("missing note must return 404")
	}
}
func TestInertiaFeatureMetadataAndSettings(t *testing.T) {
	c := newClient(t)
	c.login("demo@example.com")
	p := page(t, c.request("GET", "/", nil, nil))
	if p["deferredProps"] == nil {
		t.Fatal("deferred metadata missing")
	}
	r := c.request("GET", "/", nil, map[string]string{"X-Inertia-Partial-Component": "Dashboard", "X-Inertia-Partial-Data": "stats,recent"})
	if props(t, r)["stats"].(map[string]any)["note_count"] != float64(32) {
		t.Fatal("deferred stats not evaluated")
	}
	r = c.request("GET", "/?failStats=1", nil, map[string]string{"X-Inertia-Partial-Component": "Dashboard", "X-Inertia-Partial-Data": "stats"})
	if r.Code != 200 || page(t, r)["rescuedProps"] == nil {
		t.Fatalf("rescue response: %s", r.Body.String())
	}
	p = page(t, c.request("GET", "/notes?page=2", nil, nil))
	if p["scrollProps"] == nil {
		t.Fatal("scroll metadata missing")
	}
	data := p["props"].(map[string]any)["notes"].(map[string]any)["data"].([]any)
	if len(data) != 10 {
		t.Fatalf("second page length: %d", len(data))
	}
	p = page(t, c.request("GET", "/?merge=1", nil, map[string]string{"X-Inertia-Partial-Component": "Dashboard", "X-Inertia-Partial-Data": "recent"}))
	if p["mergeProps"] == nil || p["matchPropsOn"] == nil {
		t.Fatalf("merge metadata: %v", p)
	}
	p = page(t, c.request("GET", "/settings", nil, nil))
	if p["onceProps"] == nil {
		t.Fatal("once metadata missing")
	}
	r = c.request("POST", "/settings", map[string]any{"theme": "dark", "page_size": 25}, nil)
	if r.Code != 303 {
		t.Fatalf("save settings: %s", r.Body.String())
	}
	if got := c.db.GetSettings("u1"); got.Theme != "dark" || got.PageSize != 25 {
		t.Fatalf("saved settings: %+v", got)
	}
	p = props(t, c.request("GET", "/settings", nil, map[string]string{"X-Inertia-Except-Once-Props": "user-settings-u1"}))
	if p["settings"].(map[string]any)["theme"] != "dark" {
		t.Fatal("saved once settings were not fresh")
	}
	r = c.request("POST", "/settings", map[string]any{"theme": "invalid", "page_size": 0}, nil)
	if props(t, r)["errors"] == nil {
		t.Fatal("invalid settings accepted")
	}
}
