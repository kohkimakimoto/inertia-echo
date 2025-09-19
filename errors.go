package inertia

import (
	"errors"
)

var (
	ErrNoInertiaContext      = errors.New("inertia-echo: echo.Context does not have 'Inertia'")
	ErrRendererNotRegistered = errors.New("inertia-echo: renderer not registered")
)
