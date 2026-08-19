package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testBody = `{"hello":"world"}`

func echoHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Write(body)
	})
}

// --- GzipDecompress ---

func TestGzipDecompress_PlainBody(t *testing.T) {
	h := GzipDecompress(echoHandler())

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testBody))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != testBody {
		t.Errorf("body = %q, want %q", got, testBody)
	}
}

func TestGzipDecompress_GzipBody(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte(testBody))
	gz.Close()

	h := GzipDecompress(echoHandler())

	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if got := w.Body.String(); got != testBody {
		t.Errorf("body = %q, want %q", got, testBody)
	}
}

func TestGzipDecompress_InvalidGzip(t *testing.T) {
	h := GzipDecompress(echoHandler())

	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("not-gzip"))
	r.Header.Set("Content-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestGzipDecompress_HeaderCleared(t *testing.T) {
	var gotEncoding string
	spy := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotEncoding = r.Header.Get("Content-Encoding")
	})

	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write([]byte("x"))
	gz.Close()

	h := GzipDecompress(spy)
	r := httptest.NewRequest(http.MethodPost, "/", &buf)
	r.Header.Set("Content-Encoding", "gzip")
	h.ServeHTTP(httptest.NewRecorder(), r)

	if gotEncoding != "" {
		t.Errorf("Content-Encoding not cleared, got %q", gotEncoding)
	}
}

// --- GzipCompress ---

func TestGzipCompress_NoAcceptEncoding(t *testing.T) {
	h := GzipCompress(echoHandler())

	r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(testBody))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Header().Get("Content-Encoding") == "gzip" {
		t.Error("Content-Encoding should not be gzip when client doesn't accept it")
	}
	if got := w.Body.String(); got != testBody {
		t.Errorf("body = %q, want %q", got, testBody)
	}
}

func TestGzipCompress_WithAcceptEncoding(t *testing.T) {
	h := GzipCompress(echoHandler())

	r := httptest.NewRequest(http.MethodGet, "/", strings.NewReader(testBody))
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if enc := w.Header().Get("Content-Encoding"); enc != "gzip" {
		t.Fatalf("Content-Encoding = %q, want gzip", enc)
	}

	gr, err := gzip.NewReader(w.Body)
	if err != nil {
		t.Fatalf("gzip.NewReader: %v", err)
	}
	defer gr.Close()

	got, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != testBody {
		t.Errorf("decompressed body = %q, want %q", string(got), testBody)
	}
}

func TestGzipCompress_ContentLengthRemoved(t *testing.T) {
	fixed := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "999")
		w.Write([]byte(testBody))
	})

	h := GzipCompress(fixed)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if cl := w.Header().Get("Content-Length"); cl != "" {
		t.Errorf("Content-Length should be removed, got %q", cl)
	}
}

func TestGzipCompress_VaryHeader(t *testing.T) {
	h := GzipCompress(echoHandler())
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Accept-Encoding", "gzip")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	if vary := w.Header().Get("Vary"); vary == "" {
		t.Error("Vary header should be set")
	}
}
