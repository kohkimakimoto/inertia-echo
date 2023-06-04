package v8go

import (
	"github.com/kohkimakimoto/inertia-echo"
	"rogchap.com/v8go"
	"runtime"
	"sync"
)

type SsrEngine struct {
	Source string
	Origin string
	pool   sync.Pool
}

func NewSsrEngine() *SsrEngine {
	e := &SsrEngine{}
	e.pool.New = func() interface{} {
		// It is inspired by https://github.com/rogchap/v8go/issues/105#issuecomment-867332376
		iso := v8go.NewIsolate()
		runtime.SetFinalizer(iso, e.finalizer)
		return iso
	}
	return e
}

func (e *SsrEngine) Render(p *inertia.Page) (*inertia.SsrResponse, error) {
	return nil, nil
}

func (e *SsrEngine) finalizer(iso *v8go.Isolate) {
	iso.Dispose()
	runtime.SetFinalizer(iso, nil)
}
