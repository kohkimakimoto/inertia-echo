package inertia

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// ResponseWriterWrapper is a wrapper of http.ResponseWriter for buffering a response status code.
// Inertia.js adapter needs to change the response status code in a middleware.
// For example, if a request has X-Inertia, the adapter change the response code to 303 from 302.
// see https://inertiajs.com/redirects
type ResponseWriterWrapper struct {
	http.ResponseWriter
	buffered    bool
	wroteHeader bool
	statusCode  int
	body        bytes.Buffer
}

func NewResponseWriterWrapper(w http.ResponseWriter) *ResponseWriterWrapper {
	return &ResponseWriterWrapper{
		ResponseWriter: w,
		buffered:       false,
		wroteHeader:    false,
		statusCode:     http.StatusOK,
	}
}

// WriteHeader buffers redirect responses and forwards other status codes.
// As required by net/http, only the first call takes effect.
func (w *ResponseWriterWrapper) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.statusCode = statusCode

	if isRedirectStatus(statusCode) || (statusCode == http.StatusConflict &&
		(w.Header().Get(HeaderXInertiaLocation) != "" || w.Header().Get(HeaderXInertiaRedirect) != "")) {
		w.buffered = true
		return
	}

	// otherwise, send the header
	w.buffered = false
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *ResponseWriterWrapper) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.buffered {
		return w.body.Write(p)
	}

	return w.ResponseWriter.Write(p)
}

func (w *ResponseWriterWrapper) replaceBufferedStatusCode(from, to int) {
	if w.buffered && w.statusCode == from {
		w.statusCode = to
	}
}

func (w *ResponseWriterWrapper) replaceBufferedResponse(statusCode int, body []byte) {
	if !w.buffered {
		return
	}
	w.statusCode = statusCode
	w.body.Reset()
	if len(body) > 0 {
		_, _ = w.body.Write(body)
	}
}

func (w *ResponseWriterWrapper) hasFragmentRedirect() bool {
	return w.buffered && isRedirectStatus(w.statusCode) && strings.Contains(w.Header().Get("Location"), "#")
}

func (w *ResponseWriterWrapper) isTransitionResponse() bool {
	if !w.buffered {
		return false
	}
	return isRedirectStatus(w.statusCode) || (w.statusCode == http.StatusConflict &&
		(w.Header().Get(HeaderXInertiaLocation) != "" || w.Header().Get(HeaderXInertiaRedirect) != ""))
}

func isRedirectStatus(statusCode int) bool {
	switch statusCode {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// Unwrap returns the underlying response writer.
func (w *ResponseWriterWrapper) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

// Flush implements http.Flusher while preserving a buffered redirect response.
func (w *ResponseWriterWrapper) Flush() {
	err := w.FlushError()
	if err != nil && errors.Is(err, http.ErrNotSupported) {
		panic(fmt.Errorf("inertia-echo: response writer %T does not support flushing (http.Flusher interface)", w.ResponseWriter))
	}
}

// Hijack implements http.Hijacker by delegating to the underlying response writer.
func (w *ResponseWriterWrapper) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return http.NewResponseController(w.ResponseWriter).Hijack()
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
