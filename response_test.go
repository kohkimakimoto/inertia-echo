package inertia

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
