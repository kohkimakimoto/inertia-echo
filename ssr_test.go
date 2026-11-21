package inertia

import (
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestSsrResponse_HeadHTML(t *testing.T) {
	response := &SsrResponse{
		Head: []string{
			"<title>Test Page</title>",
			"<meta name=\"description\" content=\"test\">",
			"<link rel=\"stylesheet\" href=\"/style.css\">",
		},
	}

	result := response.HeadHTML()
	expected := "<title>Test Page</title>\n<meta name=\"description\" content=\"test\">\n<link rel=\"stylesheet\" href=\"/style.css\">"

	if string(result) != expected {
		t.Errorf("Expected HeadHTML to return '%s', got '%s'", expected, string(result))
	}
}


func TestSsrResponse_BodyHTML(t *testing.T) {
	response := &SsrResponse{
		Body: "<div><h1>Hello World</h1><p>This is a test</p></div>",
	}

	result := response.BodyHTML()
	expected := "<div><h1>Hello World</h1><p>This is a test</p></div>"

	if string(result) != expected {
		t.Errorf("Expected BodyHTML to return '%s', got '%s'", expected, string(result))
	}
}


func TestMatchSSRPath(t *testing.T) {
	tests := []struct {
		path, pattern string
		want          bool
	}{
		{"/admin/users", "admin/*", true},
		{"/admin/users", "/admin/*", true},
		{"/users", "admin/*", false},
		{"/", "/", true},
		{"/exact", "exact", true},
	}
	for _, tt := range tests {
		if got := matchSSRPath(tt.path, tt.pattern); got != tt.want {
			t.Fatalf("matchSSRPath(%q, %q)=%v, want %v", tt.path, tt.pattern, got, tt.want)
		}
	}
}


func TestMiddlewareWithoutSsr(t *testing.T) {
	e := echo.New()
	dispatched := false
	gateway := gatewayFunc(func(ctx *RenderContext) (*SsrResponse, error) {
		dispatched = true
		return &SsrResponse{Body: "<div>ssr</div>"}, nil
	})
	r := NewHTMLRenderer()
	r.SsrGateway = gateway
	r.Templates = mustParseRendererTemplate(t, r, `{{ define "app.html" }}{{ .inertia }}{{ end }}`)

	e.Use(MiddlewareWithConfig(MiddlewareConfig{
		VersionFunc: func(*echo.Context) string { return "v3" },
		WithoutSsr:  []string{"admin/*"},
		Renderer:    r,
	}))
	e.GET("/admin/users", func(c *echo.Context) error {
		if SsrEnabled(c) {
			t.Fatal("SsrEnabled should be false for WithoutSsr match")
		}
		return Render(c, http.StatusOK, "Home", nil)
	})
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/admin/users", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d", rec.Code)
	}
	if dispatched {
		t.Fatal("SsrGateway.Dispatch must not run when Inertia.SsrEnabled is false")
	}
	if !strings.Contains(rec.Body.String(), `type="application/json"`) {
		t.Fatalf("expected CSR bootstrap, got %s", rec.Body.String())
	}
}


func TestInertiaSsrEnabled(t *testing.T) {
	e := echo.New()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/admin/users", nil), httptest.NewRecorder())
	i := &Inertia{echoContext: c}
	if !i.SsrEnabled() {
		t.Fatal("default should allow SSR")
	}
	i.DisableSsr()
	if i.SsrEnabled() {
		t.Fatal("DisableSsr")
	}
	i.ssrDisabled = false
	i.WithoutSsr("admin/*")
	if i.SsrEnabled() {
		t.Fatal("WithoutSsr path match")
	}
	i.withoutSsr = nil
	i.DisableSsrWhen(func(*echo.Context) bool { return true })
	if i.SsrEnabled() {
		t.Fatal("DisableSsrWhen")
	}
}

type gatewayFunc func(*RenderContext) (*SsrResponse, error)

func (f gatewayFunc) Dispatch(ctx *RenderContext) (*SsrResponse, error) { return f(ctx) }

func mustParseRendererTemplate(t *testing.T, r *HTMLRenderer, body string) *template.Template {
	t.Helper()
	tmpl, err := template.New("").Funcs(r.FuncMap()).Parse(body)
	if err != nil {
		t.Fatal(err)
	}
	return tmpl
}
