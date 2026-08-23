package inertia

import (
	"bytes"
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
	body       bytes.Buffer
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

func (w *ResponseWriterWrapper) Write(p []byte) (int, error) {
	if w.buffered {
		return w.body.Write(p)
	}

	return w.ResponseWriter.Write(p)
}

// Unwrap returns the underlying response writer.
func (w *ResponseWriterWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// FlushError flushes a buffered response before flushing the underlying writer.
func (w *ResponseWriterWrapper) FlushError() error {
	if err := w.flushBufferedResponse(); err != nil {
		return err
	}

	return http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *ResponseWriterWrapper) FlushHeader() {
	_ = w.flushBufferedResponse()
}

func (w *ResponseWriterWrapper) flushBufferedResponse() error {
	if !w.buffered {
		return nil
	}

	w.buffered = false
	w.ResponseWriter.WriteHeader(w.statusCode)
	_, err := w.body.WriteTo(w.ResponseWriter)
	return err
}
