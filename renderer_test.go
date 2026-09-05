package inertia

import (
	"context"
	"encoding/json"
	"errors"
	"html"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v5"
)

func TestHTMLRendererRenderInertiaV3Bootstrap(t *testing.T) {
	r := NewHTMLRenderer()
	page := &Page{
		Component: "Index",
		Props: map[string]any{
			"unsafe":  "</script><script>alert(\"x\")</script>&\u2028\u2029",
			"unicode": "日本語",
		},
		URL:     "/?q=<&quote=\"",
		Version: "v3",
	}

	markup, err := r.renderInertia(page)
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	if !strings.HasPrefix(got, `<script type="application/json" data-page="app">`) {
		t.Fatalf("unexpected v3 bootstrap markup: %s", got)
	}
	if !strings.HasSuffix(got, `</script><div id="app"></div>`) {
		t.Fatalf("unexpected root element: %s", got)
	}
	if strings.Contains(got, `<div id="app" data-page=`) {
		t.Fatalf("legacy data-page attribute must not be emitted: %s", got)
	}
	if strings.Count(got, "</script>") != 1 {
		t.Fatalf("page JSON escaped the script element: %s", got)
	}

	start := strings.Index(got, ">") + 1
	end := strings.Index(got, "</script>")
	var decoded Page
	if err := json.Unmarshal([]byte(got[start:end]), &decoded); err != nil {
		t.Fatalf("script text is not JSON: %v", err)
	}
	if decoded.Props["unsafe"] != page.Props["unsafe"] || decoded.Props["unicode"] != "日本語" {
		t.Fatalf("page JSON did not round-trip: %#v", decoded.Props)
	}
}

func TestHTMLRendererEscapesContainerID(t *testing.T) {
	r := NewHTMLRenderer()
	r.ContainerId = `app"><script>alert(1)</script>`
	markup, err := r.renderInertia(&Page{Props: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	got := string(markup)
	if strings.Contains(got, `<script>alert(1)</script>`) {
		t.Fatalf("container ID was not attribute escaped: %s", got)
	}
	escaped := html.EscapeString(r.ContainerId)
	if strings.Count(got, escaped) != 2 {
		t.Fatalf("container ID must be shared by script and root div: %s", got)
	}
}

type rendererEngineFunc func(*RenderContext) (*SsrResponse, error)

func (f rendererEngineFunc) Render(ctx *RenderContext) (*SsrResponse, error) { return f(ctx) }

func newRendererTestInertia(t *testing.T) *Inertia {
	t.Helper()
	e := echo.New()
	c := e.NewContext(httptest.NewRequest("GET", "/", nil), httptest.NewRecorder())
	return &Inertia{echoContext: c}
}

func TestHTMLRendererSSRSkipAndFallback(t *testing.T) {
	tests := []struct {
		name       string
		engine     SsrEngine
		fallback   bool
		wantErr    bool
		wantReport bool
		cancel     bool
	}{
		{name: "null skip", engine: rendererEngineFunc(func(*RenderContext) (*SsrResponse, error) { return nil, nil })},
		{name: "strict error", engine: rendererEngineFunc(func(*RenderContext) (*SsrResponse, error) { return nil, errors.New("ssr failed") }), wantErr: true, wantReport: true},
		{name: "explicit fallback", engine: rendererEngineFunc(func(*RenderContext) (*SsrResponse, error) { return nil, errors.New("ssr failed") }), fallback: true, wantReport: true},
		{name: "request cancellation is not hidden", engine: rendererEngineFunc(func(*RenderContext) (*SsrResponse, error) { return nil, context.Canceled }), fallback: true, wantErr: true, wantReport: true, cancel: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			i := newRendererTestInertia(t)
			if tt.cancel {
				requestContext, cancel := context.WithCancel(i.EchoContext().Request().Context())
				cancel()
				i.EchoContext().SetRequest(i.EchoContext().Request().WithContext(requestContext))
			}
			r := NewHTMLRenderer()
			r.Vite = false
			r.SsrEngine = tt.engine
			r.SsrFallbackOnError = tt.fallback
			reported := false
			r.SsrErrorReporter = func(*RenderContext, error) { reported = true }
			r.MustParse(`{{ define "app.html" }}{{ .inertia }}{{ end }}`)
			var out strings.Builder
			err := r.Render(&RenderContext{Inertia: i, Page: &Page{Props: map[string]any{}}, ViewName: "app.html", Writer: &out})
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error: %v", err)
			}
			if reported != tt.wantReport {
				t.Fatalf("reported=%v, want %v", reported, tt.wantReport)
			}
			if !tt.wantErr && !strings.Contains(out.String(), `type="application/json"`) {
				t.Fatalf("expected CSR bootstrap fallback, got %s", out.String())
			}
		})
	}
}
