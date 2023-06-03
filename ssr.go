package inertia

import (
	"html/template"
	"strings"
)

type SsrEngine interface {
	Render(*Page) (*SsrResponse, error)
}

type SsrResponse struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

func (r *SsrResponse) HeadHTML() template.HTML {
	return template.HTML(strings.Join(r.Head, "\n"))
}

func (r *SsrResponse) BodyHTML() template.HTML {
	return template.HTML(r.Body)
}
