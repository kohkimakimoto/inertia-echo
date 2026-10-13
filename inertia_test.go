package inertia

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestRenderPreservesVaryHeader(t *testing.T) {
	const version = "test-version"

	tests := []struct {
		name           string
		inertiaRequest bool
		initialVary    []string
		want           []string
	}{
		{
			name: "response without existing Vary header",
			want: []string{"x-inertia"},
		},
		{
			name:        "HTML response",
			initialVary: []string{"Cookie, Accept-Encoding", "Origin"},
			want:        []string{"accept-encoding", "cookie", "origin", "x-inertia"},
		},
		{
			name:           "Inertia response with X-Inertia already present",
			inertiaRequest: true,
			initialVary:    []string{"Cookie, Accept-Encoding", "Origin, x-inertia"},
			want:           []string{"accept-encoding", "cookie", "origin", "x-inertia"},
		},
		{
			name:        "wildcard",
			initialVary: []string{"*"},
			want:        []string{"*"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.inertiaRequest {
				req.Header.Set(HeaderXInertia, "true")
				req.Header.Set(HeaderXInertiaVersion, version)
			}
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			m := MiddlewareWithConfig(MiddlewareConfig{
				VersionFunc: func(*echo.Context) string {
					return version
				},
				Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
					return nil
				}),
			})
			h := m(func(c *echo.Context) error {
				for _, value := range tt.initialVary {
					c.Response().Header().Add(echo.HeaderVary, value)
				}
				return Render(c, http.StatusOK, "Home", map[string]any{})
			})

			if err := h(c); err != nil {
				t.Fatal(err)
			}

			got := testVaryHeaderTokens(rec.Header())
			if !slices.Equal(got, tt.want) {
				t.Fatalf("unexpected Vary header tokens: got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRenderPageURLDefaultAndResolver(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		urlResolver URLResolverFunc
		wantURL     string
	}{
		{
			name:    "default uses request URL string",
			path:    "/users?page=2",
			wantURL: "/users?page=2",
		},
		{
			name: "custom resolver",
			path: "/users?page=2",
			urlResolver: func(c *echo.Context) string {
				return "/app" + c.Request().URL.String()
			},
			wantURL: "/app/users?page=2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			var page *Page
			e.Use(MiddlewareWithConfig(MiddlewareConfig{
				VersionFunc: func(*echo.Context) string { return "v3" },
				URLResolver: tt.urlResolver,
				Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
					page = ctx.Page
					return nil
				}),
			}))
			e.GET("/users", func(c *echo.Context) error {
				return Render(c, http.StatusOK, "Users", nil)
			})
			rec := httptest.NewRecorder()
			e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if rec.Code != http.StatusOK || page == nil {
				t.Fatalf("status=%d page=%#v", rec.Code, page)
			}
			if page.URL != tt.wantURL {
				t.Fatalf("URL=%q, want %q", page.URL, tt.wantURL)
			}
		})
	}
}

func TestSetURLResolver(t *testing.T) {
	e := echo.New()
	var page *Page
	e.Use(MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func(*echo.Context) string { return "v3" },
		Renderer: testNewMockRenderer(t, func(ctx *RenderContext) error {
			page = ctx.Page
			return nil
		}),
	}))
	e.GET("/", func(c *echo.Context) error {
		SetURLResolver(c, func(*echo.Context) string { return "/custom" })
		return Render(c, http.StatusOK, "Home", nil)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || page == nil {
		t.Fatalf("status=%d page=%#v", rec.Code, page)
	}
	if page.URL != "/custom" {
		t.Fatalf("URL=%q, want /custom", page.URL)
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

func TestPullClearHistory(t *testing.T) {
	tests := []struct {
		name  string
		field bool
		bag   map[string]string
		want  bool
	}{
		{name: "unset"},
		{name: "field only", field: true, want: true},
		{name: "state only", bag: map[string]string{StateKeyClearHistory: "true"}, want: true},
		{name: "field and state", field: true, bag: map[string]string{StateKeyClearHistory: "true"}, want: true},
		{name: "state non-true ignored", bag: map[string]string{StateKeyClearHistory: "1"}},
		{name: "state non-true with field", field: true, bag: map[string]string{StateKeyClearHistory: "yes"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Inertia{clearHistory: tt.field, state: tt.bag}
			got := i.pullClearHistory()
			if got != tt.want {
				t.Fatalf("pullClearHistory()=%v, want %v", got, tt.want)
			}
			if i.clearHistory {
				t.Fatal("field should be cleared after pull")
			}
			if _, ok := i.stateGet(StateKeyClearHistory); ok {
				t.Fatal("state key should be consumed after pull")
			}
			if i.pullClearHistory() {
				t.Fatal("second pull should be false")
			}
		})
	}
}

func TestPullPreserveFragment(t *testing.T) {
	tests := []struct {
		name  string
		field bool
		bag   map[string]string
		want  bool
	}{
		{name: "unset"},
		{name: "field only", field: true, want: true},
		{name: "state only", bag: map[string]string{StateKeyPreserveFragment: "true"}, want: true},
		{name: "field and state", field: true, bag: map[string]string{StateKeyPreserveFragment: "true"}, want: true},
		{name: "state non-true ignored", bag: map[string]string{StateKeyPreserveFragment: "1"}},
		{name: "state non-true with field", field: true, bag: map[string]string{StateKeyPreserveFragment: "yes"}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := &Inertia{preserveFragment: tt.field, state: tt.bag}
			got := i.pullPreserveFragment()
			if got != tt.want {
				t.Fatalf("pullPreserveFragment()=%v, want %v", got, tt.want)
			}
			if i.preserveFragment {
				t.Fatal("field should be cleared after pull")
			}
			if _, ok := i.stateGet(StateKeyPreserveFragment); ok {
				t.Fatal("state key should be consumed after pull")
			}
			if i.pullPreserveFragment() {
				t.Fatal("second pull should be false")
			}
		})
	}
}

func TestPullBooleanFlag(t *testing.T) {
	const key = "flag"

	tests := []struct {
		name  string
		field bool
		bag   map[string]string
		want  bool
	}{
		{name: "unset"},
		{name: "field only", field: true, want: true},
		{name: "state only", bag: map[string]string{key: "true"}, want: true},
		{name: "field and state", field: true, bag: map[string]string{key: "true"}, want: true},
		{name: "state non-true ignored", bag: map[string]string{key: "1"}},
		{name: "state non-true with field", field: true, bag: map[string]string{key: "yes"}, want: true},
		{name: "nil state with field", field: true, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			current := tt.field
			i := &Inertia{state: tt.bag}
			got := i.pullBooleanFlag(key, &current)
			if got != tt.want {
				t.Fatalf("pullBooleanFlag()=%v, want %v", got, tt.want)
			}
			if current {
				t.Fatal("current should be cleared after pull")
			}
			if _, ok := i.stateGet(key); ok {
				t.Fatal("state key should be consumed after pull")
			}
			if i.pullBooleanFlag(key, &current) {
				t.Fatal("second pull should be false")
			}
		})
	}
}

func TestSaveBooleanFlag(t *testing.T) {
	t.Run("true writes state and marks dirty", func(t *testing.T) {
		i := &Inertia{}
		i.saveBooleanFlag("flag", true)
		got, ok := i.stateGet("flag")
		if !ok || got != "true" {
			t.Fatalf("state got=%q ok=%v", got, ok)
		}
		if !i.stateDirty {
			t.Fatal("expected dirty after save")
		}
	})

	t.Run("false is a no-op", func(t *testing.T) {
		i := &Inertia{state: map[string]string{"other": "1"}}
		i.saveBooleanFlag("flag", false)
		if _, ok := i.stateGet("flag"); ok {
			t.Fatal("false must not write state")
		}
		if i.stateDirty {
			t.Fatal("false must not mark dirty")
		}
		if got, ok := i.stateGet("other"); !ok || got != "1" {
			t.Fatalf("existing state mutated: got=%q ok=%v", got, ok)
		}
	})
}

type recordingStateStore struct {
	bags []map[string]string
	err  error
}

func (s *recordingStateStore) Load(*echo.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *recordingStateStore) Save(_ *echo.Context, bag map[string]string) error {
	copied := make(map[string]string, len(bag))
	for key, value := range bag {
		copied[key] = value
	}
	s.bags = append(s.bags, copied)
	return s.err
}

func TestPersistState(t *testing.T) {
	t.Run("clean state skips Save", func(t *testing.T) {
		store := &recordingStateStore{}
		i := &Inertia{stateStore: store, flash: map[string]any{}}
		if err := i.persistState(); err != nil {
			t.Fatal(err)
		}
		if len(store.bags) != 0 {
			t.Fatalf("Save called unexpectedly: %#v", store.bags)
		}
	})

	t.Run("persists clear history and preserve fragment", func(t *testing.T) {
		store := &recordingStateStore{}
		i := &Inertia{
			stateStore:       store,
			clearHistory:     true,
			preserveFragment: true,
			flash:            map[string]any{},
		}
		if err := i.persistState(); err != nil {
			t.Fatal(err)
		}
		if len(store.bags) != 1 {
			t.Fatalf("Save count=%d, want 1", len(store.bags))
		}
		bag := store.bags[0]
		if bag[StateKeyClearHistory] != "true" || bag[StateKeyPreserveFragment] != "true" {
			t.Fatalf("bag=%v", bag)
		}
	})

	t.Run("persists unconsumed flash", func(t *testing.T) {
		store := &recordingStateStore{}
		i := &Inertia{
			stateStore: store,
			flash:      map[string]any{"message": "hi"},
		}
		if err := i.persistState(); err != nil {
			t.Fatal(err)
		}
		if len(store.bags) != 1 {
			t.Fatalf("Save count=%d, want 1", len(store.bags))
		}
		raw := store.bags[0][StateKeyFlashData]
		decoded, err := decodeFlashState(raw)
		if err != nil {
			t.Fatal(err)
		}
		if decoded["message"] != "hi" {
			t.Fatalf("flash=%v", decoded)
		}
		if len(i.flash) != 0 {
			t.Fatalf("local flash should be cleared, got %v", i.flash)
		}
	})

	t.Run("skips flash after resolveFlash", func(t *testing.T) {
		store := &recordingStateStore{}
		i := &Inertia{
			stateStore:  store,
			flash:       map[string]any{"message": "local"},
			flashLoaded: true,
		}
		if err := i.persistState(); err != nil {
			t.Fatal(err)
		}
		if len(store.bags) != 0 {
			t.Fatalf("Save called unexpectedly: %#v", store.bags)
		}
	})

	t.Run("nil StateStore still updates bag without Save", func(t *testing.T) {
		i := &Inertia{clearHistory: true, flash: map[string]any{}}
		if err := i.persistState(); err != nil {
			t.Fatal(err)
		}
		if got, ok := i.stateGet(StateKeyClearHistory); !ok || got != "true" {
			t.Fatalf("bag not updated: got=%q ok=%v", got, ok)
		}
	})

	t.Run("Save error propagates", func(t *testing.T) {
		wantErr := errors.New("persist state failed")
		store := &recordingStateStore{err: wantErr}
		i := &Inertia{stateStore: store, clearHistory: true, flash: map[string]any{}}
		if err := i.persistState(); err != wantErr {
			t.Fatalf("err=%v, want %v", err, wantErr)
		}
	})
}

func testVaryHeaderTokens(header http.Header) []string {
	var tokens []string
	for _, value := range header.Values(echo.HeaderVary) {
		for _, token := range strings.Split(value, ",") {
			token = strings.TrimSpace(token)
			if token != "" {
				tokens = append(tokens, strings.ToLower(token))
			}
		}
	}
	slices.Sort(tokens)
	return tokens
}
