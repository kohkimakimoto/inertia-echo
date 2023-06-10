package inertia

import (
	"github.com/labstack/echo/v4"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestInertia_SetRenderer(t *testing.T) {
	i := &Inertia{
		renderer: nil,
	}
	if i.Renderer() != nil {
		t.Errorf("renderer should be nil")
	}

	i.SetRenderer(testMockRenderer(t, func(w io.Writer, name string, data map[string]interface{}, in *Inertia) error {
		return nil
	}))

	if i.Renderer() == nil {
		t.Errorf("renderer should not be nil")
	}
}

func TestInertia_IsSsrDisabled(t *testing.T) {
	i := &Inertia{
		isSsrDisabled: false,
	}
	i.EnableSsr()
	if i.IsSsrDisabled() {
		t.Errorf("ssr should not be disabled")
	}
	if !i.IsSsrEnabled() {
		t.Errorf("ssr should be enabled")
	}

	i.DisableSsr()
	if !i.IsSsrDisabled() {
		t.Errorf("ssr should be disabled")
	}
	if i.IsSsrEnabled() {
		t.Errorf("ssr should not be enabled")
	}
}

func TestInertia_SetRootView(t *testing.T) {
	i := &Inertia{
		rootView: "",
	}
	if i.RootView() != "" {
		t.Errorf("root view should be empty")
	}

	i.SetRootView("app.html")
	if i.RootView() != "app.html" {
		t.Errorf("root view should be app.html")
	}
}

func TestInertia_Share(t *testing.T) {
	i := &Inertia{
		sharedProps: map[string]interface{}{},
	}
	if len(i.Shared()) != 0 {
		t.Errorf("shared props should be empty")
	}

	i.Share(map[string]interface{}{
		"foo": "bar",
	})
	if len(i.Shared()) != 1 {
		t.Errorf("shared props should have 1 item")
	}
	if i.Shared()["foo"] != "bar" {
		t.Errorf("shared props should have foo=bar")
	}

	i.FlushShared()
	if len(i.Shared()) != 0 {
		t.Errorf("shared props should be empty")
	}
}

func TestInertia_Version(t *testing.T) {
	i := &Inertia{
		version: func() string {
			return "1.0.0"
		},
	}
	if i.Version() != "1.0.0" {
		t.Errorf("version should be 1.0.0")
	}

	i.SetVersion(func() string {
		return "2.0.0"
	})
	if i.Version() != "2.0.0" {
		t.Errorf("version should be 2.0.0")
	}
}

func TestInertia_Location(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	res := httptest.NewRecorder()
	c := e.NewContext(req, res)

	i := &Inertia{
		c: c,
	}

	err := i.Location("/foo")
	if err != nil {
		t.Errorf("should not return error")
	}
	if c.Response().Header().Get(HeaderXInertiaLocation) != "/foo" {
		t.Errorf("%s should be /foo", HeaderXInertiaLocation)
	}
}
