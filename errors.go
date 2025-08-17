package inertia

import (
	"encoding/gob"
	"errors"
	"sync"
)

var (
	ErrNoInertiaContext          = errors.New("inertia-echo: echo.Context does not have 'Inertia'")
	ErrRendererNotRegistered     = errors.New("inertia-echo: renderer not registered")
	ErrSessionStoreNotRegistered = errors.New("inertia-echo: session store not registered")
)

func init() {
	// Registering is required to save cookie session.
	gob.Register(map[string]string{})
}

// ErrorMessageMap is a struct that stores multiple error messages.
// It is commonly used to collect and handle validation errors within the Inertia.js project.
// see https://inertiajs.com/validation
type ErrorMessageMap struct {
	data  map[string]string
	mutex sync.RWMutex
}

// NewErrorMessageMap creates a new ErrorMessageMap
func NewErrorMessageMap() *ErrorMessageMap {
	return &ErrorMessageMap{
		data: make(map[string]string),
	}
}

// Set sets an error message for the given key
func (e *ErrorMessageMap) Set(key, errMessage string) *ErrorMessageMap {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.data[key] = errMessage

	return e
}

// Get gets an error message for the given key
func (e *ErrorMessageMap) Get(key string) (string, bool) {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	value, exists := e.data[key]
	return value, exists
}

// Update updates the error messages with the provided map
func (e *ErrorMessageMap) Update(errMessages map[string]string) *ErrorMessageMap {
	e.mutex.Lock()
	defer e.mutex.Unlock()

	for key, value := range errMessages {
		e.data[key] = value
	}

	return e
}

// Len returns the number of error messages
func (e *ErrorMessageMap) Len() int {
	e.mutex.RLock()
	defer e.mutex.RUnlock()
	return len(e.data)
}

// ToMap converts the ErrorMessageMap to a standard map
func (e *ErrorMessageMap) ToMap() map[string]string {
	e.mutex.RLock()
	defer e.mutex.RUnlock()

	result := make(map[string]string, len(e.data))
	for key, value := range e.data {
		result[key] = value
	}

	return result
}

// Clear clears all error messages
func (e *ErrorMessageMap) Clear() *ErrorMessageMap {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.data = make(map[string]string)

	return e
}
