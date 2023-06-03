package inertia

import "errors"

var (
	ErrNotFound              = errors.New("inertia-echo: context does not have 'Inertia'")
	ErrRendererNotRegistered = errors.New("inertia-echo: renderer not registered")
)
