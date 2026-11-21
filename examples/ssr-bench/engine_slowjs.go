package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	"github.com/liustack/slowjs"
	"github.com/liustack/slowjs/intl"
	"github.com/liustack/slowjs/temporal"
	"github.com/liustack/slowjs/url"
)

//go:embed polyfill.js
var polyfillJS string

type slowjsEngine struct {
	root   string
	bundle string
	vm     *slowjs.VM
}

func newSlowjsEngine(root, bundle string) *slowjsEngine {
	return &slowjsEngine{root: root, bundle: bundle}
}

func (e *slowjsEngine) Name() string { return "slowjs" }

func (e *slowjsEngine) Open() error {
	src, err := os.ReadFile(e.bundle)
	if err != nil {
		return fmt.Errorf("read bundle: %w (run: make -C ../ssr-slowjs ssr-bundle)", err)
	}
	vm, err := slowjs.NewVM()
	if err != nil {
		return err
	}
	vm.SetMemoryLimit(128 << 20)
	if err := vm.SetEvalTimeout(10 * time.Second); err != nil {
		vm.Close()
		return err
	}
	if err := url.Register(vm); err != nil {
		vm.Close()
		return fmt.Errorf("url.Register: %w", err)
	}
	if err := intl.Register(vm); err != nil {
		vm.Close()
		return fmt.Errorf("intl.Register: %w", err)
	}
	if err := temporal.Register(vm); err != nil {
		vm.Close()
		return fmt.Errorf("temporal.Register: %w", err)
	}
	if _, err := vm.Eval(polyfillJS, slowjs.EvalGlobal); err != nil {
		vm.Close()
		return fmt.Errorf("polyfill: %w", err)
	}
	if _, err := vm.Eval(string(src), slowjs.EvalGlobal); err != nil {
		vm.Close()
		return fmt.Errorf("bundle: %w", err)
	}
	e.vm = vm
	return nil
}

func (e *slowjsEngine) Close() {
	if e.vm != nil {
		e.vm.Close()
		e.vm = nil
	}
}

func (e *slowjsEngine) Render(page *inertia.Page) (*inertia.SsrResponse, error) {
	raw, err := e.vm.Call("renderPage", page)
	if err != nil {
		return nil, err
	}
	obj, ok := raw.(*slowjs.Object)
	if !ok {
		return nil, fmt.Errorf("renderPage returned %T", raw)
	}
	var out struct {
		Head []string `json:"head"`
		Body string   `json:"body"`
	}
	if err := obj.Into(&out); err != nil {
		return nil, err
	}
	return &inertia.SsrResponse{Head: out.Head, Body: out.Body}, nil
}

func (e *slowjsEngine) ExtraRSSBytes() uint64 { return 0 }

// ensure bundle path default
func defaultSlowjsBundle(root string) string {
	return filepath.Join(root, "..", "ssr-slowjs", ".build", "ssr", "ssr.js")
}
