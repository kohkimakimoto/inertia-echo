package inertia

import (
	"net/http"
)

// ResponseWriterWrapper is a wrapper of http.ResponseWriter for buffering a response status code.
// Inertia.js adapter needs to change the response status code in a middleware.
// For example, if a request has X-Inertia, the adapter change the response code to 303 from 302.
// see https://inertiajs.com/redirects
type ResponseWriterWrapper struct {
	http.ResponseWriter
	buffered   bool
	statusCode int
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		buffered:       false,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader stores header instead of sending it, if it is not 200
func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	if statusCode == 302 || statusCode == 303 {
		// buffering only 302 or 303 status. it is current Inertia.js protocol specification.
		// see also https://inertiajs.com/redirects
		w.buffered = true
		w.statusCode = statusCode
		return
	}

	// otherwise, send the header
	w.buffered = false
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ResponseWriterWrapper) FlushHeader() {
	if w.buffered {
		w.ResponseWriter.WriteHeader(w.statusCode)
	}
}
