package slowjs

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	slowjsvm "github.com/liustack/slowjs"
	"github.com/liustack/slowjs/intl"
	"github.com/liustack/slowjs/temporal"
	"github.com/liustack/slowjs/url"
)

//go:embed polyfill.js
var defaultPolyfillJS string

// Gateway runs a thin Inertia SSR IIFE inside a pool of slowjs VMs.
// It implements [inertia.SsrGateway] without Node or HTTP.
//
// A slowjs VM is not safe for concurrent goroutines. The gateway keeps a fixed
// pool and hands out one VM per Dispatch via a buffered channel (acquire /
// release). PoolSize 1 matches a single locked VM: at most one render at a time.
type Gateway struct {
	cfg    Config
	bundle string

	pool     chan *worker
	closed   chan struct{}
	inflight sync.WaitGroup
	close    sync.Once
}

type worker struct {
	vm *slowjsvm.VM
}

type ssrResult struct {
	Head []string `json:"head"`
	Body string   `json:"body"`
}

// Config configures [New] / [NewFromSource]. Zero values mean defaults.
type Config struct {
	// MemoryLimit is the slowjs heap limit in bytes per VM. Default: 128 MiB.
	MemoryLimit uintptr
	// EvalTimeout bounds a single Eval/Call. Default: 5s.
	EvalTimeout time.Duration
	// Polyfill is JavaScript evaluated before the SSR bundle.
	// Empty uses the embedded React/Inertia host stubs.
	Polyfill string
	// PoolSize is the number of warm VMs. Default: max(1, GOMAXPROCS).
	//
	//	1  — lowest memory; serial renders (same concurrency as the old mutex VM)
	//	N  — up to N concurrent renders; typical choice is GOMAXPROCS for CPU-bound SSR
	PoolSize int
}

func (c Config) withDefaults() Config {
	if c.MemoryLimit == 0 {
		c.MemoryLimit = 128 << 20
	}
	if c.EvalTimeout == 0 {
		c.EvalTimeout = 5 * time.Second
	}
	if c.Polyfill == "" {
		c.Polyfill = defaultPolyfillJS
	}
	if c.PoolSize <= 0 {
		c.PoolSize = runtime.GOMAXPROCS(0)
	}
	if c.PoolSize < 1 {
		c.PoolSize = 1
	}
	return c
}

// New loads an SSR IIFE from bundlePath into a pool of slowjs VMs.
func New(bundlePath string, cfg ...Config) (*Gateway, error) {
	src, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("ssr/slowjs: read bundle: %w", err)
	}
	return NewFromSource(string(src), cfg...)
}

// NewFromSource evaluates bundleJS (IIFE source) in a pool of slowjs VMs.
func NewFromSource(bundleJS string, cfg ...Config) (*Gateway, error) {
	var c Config
	if len(cfg) > 0 {
		c = cfg[0]
	}
	c = c.withDefaults()

	g := &Gateway{
		cfg:    c,
		bundle: bundleJS,
		pool:   make(chan *worker, c.PoolSize),
		closed: make(chan struct{}),
	}
	for i := 0; i < c.PoolSize; i++ {
		w, err := g.newWorker()
		if err != nil {
			g.Close()
			return nil, err
		}
		g.pool <- w
	}
	return g, nil
}

func (g *Gateway) newWorker() (*worker, error) {
	vm, err := slowjsvm.NewVM()
	if err != nil {
		return nil, fmt.Errorf("ssr/slowjs: new vm: %w", err)
	}
	vm.SetMemoryLimit(g.cfg.MemoryLimit)
	if err := vm.SetEvalTimeout(g.cfg.EvalTimeout); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: set timeout: %w", err)
	}

	// Go-implemented Web APIs (same building blocks slowjs/tifa.Register uses,
	// without tifa's React acceleration hooks). Order matches tifa.Register.
	if err := url.Register(vm); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: url.Register: %w", err)
	}
	if err := intl.Register(vm); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: intl.Register: %w", err)
	}
	if err := temporal.Register(vm); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: temporal.Register: %w", err)
	}
	if _, err := vm.Eval(g.cfg.Polyfill, slowjsvm.EvalGlobal); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: eval polyfill: %w", err)
	}
	if _, err := vm.Eval(g.bundle, slowjsvm.EvalGlobal); err != nil {
		vm.Close()
		return nil, fmt.Errorf("ssr/slowjs: eval bundle: %w", err)
	}
	return &worker{vm: vm}, nil
}

func (w *worker) close() {
	if w != nil && w.vm != nil {
		w.vm.Close()
		w.vm = nil
	}
}

// Close waits for in-flight renders, then closes every VM. Safe to call more than once.
func (g *Gateway) Close() {
	if g == nil {
		return
	}
	g.close.Do(func() {
		close(g.closed)
		g.inflight.Wait()
		close(g.pool)
		for w := range g.pool {
			w.close()
		}
	})
}

func (g *Gateway) acquire(ctx context.Context) (*worker, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-g.closed:
		return nil, fmt.Errorf("ssr/slowjs: gateway closed")
	default:
	}

	g.inflight.Add(1)
	select {
	case w, ok := <-g.pool:
		if !ok || w == nil {
			g.inflight.Done()
			return nil, fmt.Errorf("ssr/slowjs: gateway closed")
		}
		return w, nil
	case <-ctx.Done():
		g.inflight.Done()
		return nil, fmt.Errorf("ssr/slowjs: wait for vm: %w", ctx.Err())
	case <-g.closed:
		g.inflight.Done()
		return nil, fmt.Errorf("ssr/slowjs: gateway closed")
	}
}

func (g *Gateway) release(w *worker) {
	defer g.inflight.Done()
	if w == nil {
		return
	}
	select {
	case <-g.closed:
		w.close()
	default:
		select {
		case <-g.closed:
			w.close()
		case g.pool <- w:
		}
	}
}

func requestContext(ctx *inertia.RenderContext) context.Context {
	if ctx != nil && ctx.Inertia != nil {
		if ec := ctx.Inertia.EchoContext(); ec != nil && ec.Request() != nil {
			return ec.Request().Context()
		}
	}
	return context.Background()
}

// Dispatch renders the Inertia page to HTML via globalThis.renderPage.
func (g *Gateway) Dispatch(ctx *inertia.RenderContext) (*inertia.SsrResponse, error) {
	if g == nil {
		return nil, fmt.Errorf("ssr/slowjs: nil gateway")
	}
	if ctx == nil || ctx.Page == nil {
		return nil, fmt.Errorf("ssr/slowjs: render context or page is nil")
	}
	if ctx.Inertia != nil && !ctx.Inertia.SsrEnabled() {
		return nil, nil
	}

	w, err := g.acquire(requestContext(ctx))
	if err != nil {
		return nil, err
	}
	defer g.release(w)

	raw, err := w.vm.Call("renderPage", ctx.Page)
	if err != nil {
		// A killed VM is only good for Close; replace it before returning to the pool.
		if errors.Is(err, slowjsvm.ErrKilled) {
			w.close()
			nw, nerr := g.newWorker()
			if nerr != nil {
				w = nil // release only Done(); do not put a dead VM back
				return nil, fmt.Errorf("ssr/slowjs: renderPage: %w (replace vm: %v)", err, nerr)
			}
			w = nw
		}
		return nil, fmt.Errorf("ssr/slowjs: renderPage: %w", err)
	}
	obj, ok := raw.(*slowjsvm.Object)
	if !ok {
		return nil, fmt.Errorf("ssr/slowjs: renderPage returned %T, want object", raw)
	}

	var out ssrResult
	if err := obj.Into(&out); err != nil {
		return nil, fmt.Errorf("ssr/slowjs: decode result: %w", err)
	}
	if out.Body == "" {
		return nil, fmt.Errorf("ssr/slowjs: empty body")
	}
	return &inertia.SsrResponse{Head: out.Head, Body: out.Body}, nil
}

// PoolSize returns the configured number of VMs.
func (g *Gateway) PoolSize() int {
	if g == nil {
		return 0
	}
	return g.cfg.PoolSize
}

// Compile-time check.
var _ inertia.SsrGateway = (*Gateway)(nil)
