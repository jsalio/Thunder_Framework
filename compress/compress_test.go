package compress

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func gzipMiddleware(handler http.Handler) http.Handler {
	mw := Gzip()
	return mw(handler)
}

func readGzipBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		t.Fatalf("failed to create gzip reader: %v", err)
	}
	defer gr.Close()
	b, err := io.ReadAll(gr)
	if err != nil {
		t.Fatalf("failed to read gzip body: %v", err)
	}
	return string(b)
}

func TestGzipCompressesHTMLResponse(t *testing.T) {
	body := strings.Repeat("<h1>Hello Thunder</h1>", 50)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatalf("expected Content-Encoding: gzip, got %q", resp.Header.Get("Content-Encoding"))
	}
	if resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("expected Vary: Accept-Encoding, got %q", resp.Header.Get("Vary"))
	}

	got := readGzipBody(t, resp)
	if got != body {
		t.Fatalf("decompressed body mismatch: got %d bytes, want %d bytes", len(got), len(body))
	}
}

func TestGzipSkipsSmallResponses(t *testing.T) {
	body := "hi"
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("small response should not be compressed")
	}

	b, _ := io.ReadAll(resp.Body)
	if string(b) != body {
		t.Fatalf("body mismatch: got %q, want %q", string(b), body)
	}
}

func TestGzipSkipsNonCompressibleContentType(t *testing.T) {
	body := strings.Repeat("binary data", 100)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("image/png should not be compressed")
	}
}

func TestGzipAlwaysSetsVaryHeader(t *testing.T) {
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte("hello"))
	}))

	// Request without gzip support.
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Vary") != "Accept-Encoding" {
		t.Fatalf("expected Vary: Accept-Encoding even without gzip, got %q", resp.Header.Get("Vary"))
	}
	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("should not compress when client doesn't accept gzip")
	}
}

func TestGzipSkipsWhenContentEncodingAlreadySet(t *testing.T) {
	body := strings.Repeat("already encoded", 100)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Encoding", "br")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") != "br" {
		t.Fatalf("expected original Content-Encoding: br, got %q", resp.Header.Get("Content-Encoding"))
	}

	b, _ := io.ReadAll(resp.Body)
	if string(b) != body {
		t.Fatal("body should pass through unchanged")
	}
}

func TestGzipSkips204NoContent(t *testing.T) {
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("204 should not be compressed")
	}
}

func TestGzipCompressesJSON(t *testing.T) {
	body := strings.Repeat(`{"key":"value"}`, 50)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") != "gzip" {
		t.Fatal("JSON should be compressed")
	}

	got := readGzipBody(t, resp)
	if got != body {
		t.Fatal("decompressed JSON mismatch")
	}
}

func TestGzipSkipsUpgradeRequests(t *testing.T) {
	body := strings.Repeat("websocket data", 100)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Upgrade", "websocket")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Encoding") == "gzip" {
		t.Fatal("upgrade requests should not be compressed")
	}
}

func TestAcceptsGzip(t *testing.T) {
	tests := []struct {
		header string
		want   bool
	}{
		{"gzip", true},
		{"gzip, deflate, br", true},
		{"deflate, gzip;q=1.0", true},
		{"deflate", false},
		{"", false},
		{"gzip;q=0", false},
		{"gzip;q=0.5", true},
		{"br, identity", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest("GET", "/", nil)
		req.Header.Set("Accept-Encoding", tt.header)
		got := AcceptsGzip(req)
		if got != tt.want {
			t.Errorf("AcceptsGzip(%q) = %v, want %v", tt.header, got, tt.want)
		}
	}
}

func TestGzipContentLengthRemoved(t *testing.T) {
	body := strings.Repeat("remove content length", 50)
	handler := gzipMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", "1000")
		w.Write([]byte(body))
	}))

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	if resp.Header.Get("Content-Length") != "" {
		t.Fatalf("Content-Length should be removed when compressing, got %q", resp.Header.Get("Content-Length"))
	}
}
