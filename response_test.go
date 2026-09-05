package inertia

import (
	"bufio"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	_ http.Flusher  = (*ResponseWriterWrapper)(nil)
	_ http.Hijacker = (*ResponseWriterWrapper)(nil)
)

type basicResponseWriter struct {
	header http.Header
}

type failingResponseWriter struct {
	*basicResponseWriter
	err error
}

func (w *failingResponseWriter) Write([]byte) (int, error) {
	return 0, w.err
}

func newBasicResponseWriter() *basicResponseWriter {
	return &basicResponseWriter{header: make(http.Header)}
}

func (w *basicResponseWriter) Header() http.Header {
	return w.header
}

func (w *basicResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (w *basicResponseWriter) WriteHeader(statusCode int) {}

type hijackableResponseWriter struct {
	*basicResponseWriter
	conn     net.Conn
	hijacked bool
}

func (w *hijackableResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	w.hijacked = true
	return w.conn, bufio.NewReadWriter(bufio.NewReader(w.conn), bufio.NewWriter(w.conn)), nil
}

func TestResponseWriterWrapper_PreservesRedirectStatusWhenBodyIsWritten(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := NewResponseWriterWrapper(rec)
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	http.Redirect(wrapper, req, "/next", http.StatusFound)
	wrapper.FlushHeader()
	wrapper.FlushHeader()

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, res.StatusCode)
	}
	if location := res.Header.Get("Location"); location != "/next" {
		t.Errorf("expected Location header %q, got %q", "/next", location)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if count := strings.Count(string(body), "/next"); count != 1 {
		t.Errorf("expected redirect body to contain the target once, got %d occurrences in %q", count, body)
	}
}

func TestResponseWriterWrapper_FirstWriteHeaderWins(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := NewResponseWriterWrapper(rec)

	wrapper.WriteHeader(http.StatusFound)
	if _, err := wrapper.Write([]byte("redirect body")); err != nil {
		t.Fatal(err)
	}
	wrapper.WriteHeader(http.StatusNotFound)
	wrapper.FlushHeader()

	res := rec.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound {
		t.Errorf("expected the first status code %d, got %d", http.StatusFound, res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "redirect body" {
		t.Errorf("expected body %q, got %q", "redirect body", body)
	}
}

func TestResponseWriterWrapper_ResponseControllerFlush(t *testing.T) {
	t.Run("delegates to a supported writer", func(t *testing.T) {
		rec := httptest.NewRecorder()
		wrapper := NewResponseWriterWrapper(rec)

		if err := http.NewResponseController(wrapper).Flush(); err != nil {
			t.Fatalf("expected flush to succeed, got %v", err)
		}
		if !rec.Flushed {
			t.Error("expected the underlying writer to be flushed")
		}
	})

	t.Run("flushes a buffered redirect before delegating", func(t *testing.T) {
		rec := httptest.NewRecorder()
		wrapper := NewResponseWriterWrapper(rec)
		wrapper.WriteHeader(http.StatusFound)
		if _, err := wrapper.Write([]byte("redirect body")); err != nil {
			t.Fatal(err)
		}

		if err := http.NewResponseController(wrapper).Flush(); err != nil {
			t.Fatalf("expected flush to succeed, got %v", err)
		}

		res := rec.Result()
		defer res.Body.Close()
		if res.StatusCode != http.StatusFound {
			t.Errorf("expected status code %d, got %d", http.StatusFound, res.StatusCode)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatal(err)
		}
		if string(body) != "redirect body" {
			t.Errorf("expected buffered body %q, got %q", "redirect body", body)
		}
	})

	t.Run("returns ErrNotSupported for an unsupported writer", func(t *testing.T) {
		wrapper := NewResponseWriterWrapper(newBasicResponseWriter())

		err := http.NewResponseController(wrapper).Flush()
		if !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("expected ErrNotSupported, got %v", err)
		}
	})
}

func TestResponseWriterWrapper_ResponseControllerHijack(t *testing.T) {
	t.Run("delegates to a supported writer", func(t *testing.T) {
		serverConn, clientConn := net.Pipe()
		t.Cleanup(func() {
			serverConn.Close()
			clientConn.Close()
		})

		underlying := &hijackableResponseWriter{
			basicResponseWriter: newBasicResponseWriter(),
			conn:                serverConn,
		}
		wrapper := NewResponseWriterWrapper(underlying)

		conn, _, err := http.NewResponseController(wrapper).Hijack()
		if err != nil {
			t.Fatalf("expected hijack to succeed, got %v", err)
		}
		if conn != serverConn {
			t.Error("expected the underlying connection")
		}
		if !underlying.hijacked {
			t.Error("expected the underlying writer to be hijacked")
		}
	})

	t.Run("returns ErrNotSupported for an unsupported writer", func(t *testing.T) {
		wrapper := NewResponseWriterWrapper(newBasicResponseWriter())

		_, _, err := http.NewResponseController(wrapper).Hijack()
		if !errors.Is(err, http.ErrNotSupported) {
			t.Fatalf("expected ErrNotSupported, got %v", err)
		}
	})
}

func TestResponseWriterWrapper_DirectFlush(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := NewResponseWriterWrapper(rec)
	wrapper.WriteHeader(http.StatusFound)

	wrapper.Flush()

	if !rec.Flushed {
		t.Error("expected the underlying writer to be flushed")
	}
	if rec.Code != http.StatusFound {
		t.Errorf("expected status code %d, got %d", http.StatusFound, rec.Code)
	}
}

func TestResponseWriterWrapper_DirectFlushPanicsWhenUnsupported(t *testing.T) {
	wrapper := NewResponseWriterWrapper(newBasicResponseWriter())
	defer func() {
		if recover() == nil {
			t.Fatal("expected Flush to panic for an unsupported response writer")
		}
	}()

	wrapper.Flush()
}

func TestResponseWriterWrapper_DirectHijack(t *testing.T) {
	serverConn, clientConn := net.Pipe()
	t.Cleanup(func() {
		serverConn.Close()
		clientConn.Close()
	})

	underlying := &hijackableResponseWriter{
		basicResponseWriter: newBasicResponseWriter(),
		conn:                serverConn,
	}
	wrapper := NewResponseWriterWrapper(underlying)

	conn, _, err := wrapper.Hijack()
	if err != nil {
		t.Fatalf("expected hijack to succeed, got %v", err)
	}
	if conn != serverConn {
		t.Error("expected the underlying connection")
	}
	if !underlying.hijacked {
		t.Error("expected the underlying writer to be hijacked")
	}
}

func TestResponseWriterWrapper_FlushHeader_WhenBuffered(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := NewResponseWriterWrapper(rec)

	// Set a buffered status code
	wrapper.WriteHeader(302)

	// Verify it's buffered
	if !wrapper.buffered {
		t.Fatal("expected status to be buffered")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("expected underlying recorder to still have default status %d, got %d", http.StatusOK, rec.Code)
	}

	// Flush the header
	wrapper.FlushHeader()

	// Now the underlying response writer should have the status code
	if rec.Code != 302 {
		t.Errorf("expected underlying recorder code to be 302 after flush, got %d", rec.Code)
	}
}

func TestResponseWriterWrapper_FlushHeader_WhenNotBuffered(t *testing.T) {
	rec := httptest.NewRecorder()
	wrapper := NewResponseWriterWrapper(rec)

	// Set a non-buffered status code
	wrapper.WriteHeader(404)

	// Verify it's not buffered and was written immediately
	if wrapper.buffered {
		t.Error("expected status to not be buffered")
	}
	if rec.Code != 404 {
		t.Errorf("expected underlying recorder code to be 404, got %d", rec.Code)
	}

	// Flush should be a no-op
	wrapper.FlushHeader()

	// Status should remain the same
	if rec.Code != 404 {
		t.Errorf("expected underlying recorder code to remain 404 after flush, got %d", rec.Code)
	}
}

func TestResponseWriterWrapper_FlushBufferedResponseReturnsWriteError(t *testing.T) {
	wantErr := errors.New("write failed")
	wrapper := NewResponseWriterWrapper(&failingResponseWriter{
		basicResponseWriter: newBasicResponseWriter(),
		err:                 wantErr,
	})
	wrapper.WriteHeader(http.StatusFound)
	if _, err := wrapper.Write([]byte("redirect body")); err != nil {
		t.Fatal(err)
	}

	if err := wrapper.flushBufferedResponse(); !errors.Is(err, wantErr) {
		t.Fatalf("expected %v, got %v", wantErr, err)
	}
}
