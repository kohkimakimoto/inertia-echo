package main

import (
	"path/filepath"

	inertia "github.com/kohkimakimoto/inertia-echo/v5"
	ssrslowjs "github.com/kohkimakimoto/inertia-echo/v5/ssr/slowjs"
)

type slowjsEngine struct {
	root     string
	bundle   string
	poolSize int
	gateway  *ssrslowjs.Gateway
}

func newSlowjsEngine(root, bundle string, poolSize int) *slowjsEngine {
	return &slowjsEngine{root: root, bundle: bundle, poolSize: poolSize}
}

func (e *slowjsEngine) Name() string { return "slowjs" }

func (e *slowjsEngine) Open() error {
	cfg := ssrslowjs.Config{}
	if e.poolSize > 0 {
		cfg.PoolSize = e.poolSize
	}
	g, err := ssrslowjs.New(e.bundle, cfg)
	if err != nil {
		return err
	}
	e.gateway = g
	return nil
}

func (e *slowjsEngine) Close() {
	if e.gateway != nil {
		e.gateway.Close()
		e.gateway = nil
	}
}

func (e *slowjsEngine) Render(page *inertia.Page) (*inertia.SsrResponse, error) {
	return e.gateway.Dispatch(&inertia.RenderContext{Page: page})
}

func (e *slowjsEngine) ExtraRSSBytes() uint64 { return 0 }

func defaultSlowjsBundle(root string) string {
	return filepath.Join(root, "..", "ssr-slowjs", ".build", "ssr", "ssr.js")
}
