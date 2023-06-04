package v8go

import (
	"github.com/kohkimakimoto/inertia-echo"
	"rogchap.com/v8go"
)

type SsrEngine struct {
}

func NewSsrEngine() *SsrEngine {
	return &SsrEngine{}
}

func (e *SsrEngine) Render(p *inertia.Page) (*inertia.SsrResponse, error) {
	return nil, nil
}

func (e *SsrEngine) newIsolate() *v8go.Isolate {
	return v8go.NewIsolate()
}
