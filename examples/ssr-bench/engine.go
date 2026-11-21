package main

import inertia "github.com/kohkimakimoto/inertia-echo/v5"

// Engine hosts one SSR backend for fair comparison.
type Engine interface {
	Name() string
	// Open prepares the backend (load bundle / start Node SSR). Must Close.
	Open() error
	Close()
	Render(page *inertia.Page) (*inertia.SsrResponse, error)
	// ExtraRSSBytes is optional RSS of a helper process (e.g. Node SSR). 0 if none.
	ExtraRSSBytes() uint64
}
