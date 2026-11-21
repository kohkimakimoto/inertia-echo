package main

import (
	_ "embed"
	"fmt"
	"os"
	"sync"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/liustack/slowjs"
	"github.com/liustack/slowjs/tifa"
)

//go:embed polyfill.js
var polyfillJS string


// SlowjsSsrGateway runs the thin Inertia SSR bundle inside the Go process.
// It implements inertia.SsrGateway without Node or HTTP.
type SlowjsSsrGateway struct {
	mu     sync.Mutex
	vm     *slowjs.VM
	source string
}

type ssrResult struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

// NewSlowjsSsrGateway loads bundlePath into a slowjs VM.
func NewSlowjsSsrGateway(bundlePath string) (*SlowjsSsrGateway, error) {
	src, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("slowjs ssr: read bundle: %w", err)
	}

	vm, err := slowjs.NewVM()
	if err != nil {
		return nil, fmt.Errorf("slowjs ssr: new vm: %w", err)
	}
	vm.SetMemoryLimit(128 << 20)
	if err := vm.SetEvalTimeout(5 * time.Second); err != nil {
		vm.Close()
		return nil, fmt.Errorf("slowjs ssr: set timeout: %w", err)
	}
	// React-oriented host APIs (Intl/URL/Temporal + render hot paths).
	if err := tifa.Register(vm); err != nil {
		vm.Close()
		return nil, fmt.Errorf("slowjs ssr: tifa.Register: %w", err)
	}
	if _, err := vm.Eval(polyfillJS, slowjs.EvalGlobal); err != nil {
		vm.Close()
		return nil, fmt.Errorf("slowjs ssr: eval polyfill: %w", err)
	}
	if _, err := vm.Eval(string(src), slowjs.EvalGlobal); err != nil {
		vm.Close()
		return nil, fmt.Errorf("slowjs ssr: eval bundle: %w", err)
	}

	return &SlowjsSsrGateway{vm: vm, source: bundlePath}, nil
}

// Close releases the VM.
func (g *SlowjsSsrGateway) Close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.vm != nil {
		g.vm.Close()
		g.vm = nil
	}
}

// Dispatch renders the Inertia page to HTML via globalThis.renderPage.
func (g *SlowjsSsrGateway) Dispatch(ctx *inertia.RenderContext) (*inertia.SsrResponse, error) {
	if ctx == nil || ctx.Page == nil {
		return nil, fmt.Errorf("slowjs ssr: render context or page is nil")
	}
	if ctx.Inertia != nil && !ctx.Inertia.SsrEnabled() {
		return nil, nil
	}

	g.mu.Lock()
	defer g.mu.Unlock()
	if g.vm == nil {
		return nil, fmt.Errorf("slowjs ssr: gateway closed")
	}

	raw, err := g.vm.Call("renderPage", ctx.Page)
	if err != nil {
		return nil, fmt.Errorf("slowjs ssr: renderPage: %w", err)
	}
	obj, ok := raw.(*slowjs.Object)
	if !ok {
		return nil, fmt.Errorf("slowjs ssr: renderPage returned %T, want object", raw)
	}

	var out ssrResult
	if err := obj.Into(&out); err != nil {
		return nil, fmt.Errorf("slowjs ssr: decode result: %w", err)
	}
	if out.Body == "" {
		return nil, fmt.Errorf("slowjs ssr: empty body")
	}
	return &inertia.SsrResponse{Head: out.Head, Body: out.Body}, nil
}
