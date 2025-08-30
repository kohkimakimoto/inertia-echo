package viewkitext

import (
	"encoding/json"
	"fmt"
	"github.com/kohkimakimoto/echo-viewkit"
	"github.com/kohkimakimoto/echo-viewkit/pongo2"
	"github.com/kohkimakimoto/inertia-echo/v2"
	"html/template"
	"strings"
)

type ViewKitRenderer struct {
	r           *viewkit.Renderer
	ContainerId string
	SsrEngine   inertia.SsrEngine
}

func NewRenderer(r *viewkit.Renderer) *ViewKitRenderer {
	return &ViewKitRenderer{
		r:           r,
		ContainerId: "app",
		SsrEngine:   nil,
	}
}

func (r *ViewKitRenderer) Render(ctx *inertia.RenderContext) error {
	data, err := pongo2.MarshalContext(ctx.ViewData)
	if err != nil {
		return fmt.Errorf("unsupported ViewData type: %w", err)
	}

	data["page"] = ctx.Page

	if ctx.Inertia.IsSsrEnabled() && r.SsrEngine != nil {
		// server-side rendering
		ssr, err := r.SsrEngine.Render(ctx)
		if err != nil {
			return err
		}
		data["inertia"] = pongo2.AsSafeValue(ssr.BodyHTML())
		data["inertiaHead"] = pongo2.AsSafeValue(ssr.HeadHTML())
	} else {
		// client-side rendering
		_inertia, err := r.renderInertia(ctx.Page)
		if err != nil {
			return err
		}
		data["inertia"] = pongo2.AsSafeValue(_inertia)
		data["inertiaHead"] = ""
	}

	return r.r.Render(ctx.Writer, ctx.ViewName, data, ctx.Inertia.EchoContext())
}

func (r *ViewKitRenderer) Internal() *viewkit.Renderer {
	return r.r
}

func (r *ViewKitRenderer) renderInertia(page *inertia.Page) (template.HTML, error) {
	pageJson, err := json.Marshal(page)
	if err != nil {
		return "", err
	}
	builder := new(strings.Builder)
	builder.WriteString(`<div id="` + r.ContainerId + `" data-page="`)
	template.HTMLEscape(builder, pageJson)
	builder.WriteString(`"></div>`)

	return template.HTML(builder.String()), nil
}
