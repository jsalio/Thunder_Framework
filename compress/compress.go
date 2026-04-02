// Package compress provides gzip compression middleware for the Thunder framework.
package compress

import (
	"bufio"
	"compress/gzip"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"sync"
	"thunder/router"
)

// Config holds gzip compression settings.
type Config struct {
	// Level is the gzip compression level (1-9, or gzip.DefaultCompression).
	// Default: gzip.DefaultCompression (6).
	Level int

	// MinSize is the minimum response size in bytes before compression kicks in.
	// Responses smaller than this are sent uncompressed.
	// Default: 256.
	MinSize int

	// ContentTypes is the set of Content-Type media types eligible for compression.
	// If nil, a default set is used.
	ContentTypes []string
}

var defaultCompressibleTypes = map[string]bool{
	"text/html":                true,
	"text/css":                 true,
	"text/plain":               true,
	"text/xml":                 true,
	"text/javascript":          true,
	"application/javascript":   true,
	"application/json":         true,
	"application/xml":          true,
	"application/xhtml+xml":    true,
	"application/rss+xml":      true,
	"application/atom+xml":     true,
	"application/wasm":         true,
	"image/svg+xml":            true,
}

// AcceptsGzip returns true if the request accepts gzip encoding.
func AcceptsGzip(r *http.Request) bool {
	for _, part := range strings.Split(r.Header.Get("Accept-Encoding"), ",") {
		part = strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			// Check for explicit q=0 (disabled).
			if strings.Contains(part, "q=0") && !strings.Contains(part, "q=0.") {
				return false
			}
			return true
		}
	}
	return false
}

// Gzip returns a middleware that compresses responses with gzip encoding.
func Gzip(cfgs ...Config) router.Middleware {
	cfg := Config{
		Level:   gzip.DefaultCompression,
		MinSize: 256,
	}
	if len(cfgs) > 0 {
		c := cfgs[0]
		if c.Level != 0 {
			cfg.Level = c.Level
		}
		if c.MinSize != 0 {
			cfg.MinSize = c.MinSize
		}
		if c.ContentTypes != nil {
			cfg.ContentTypes = c.ContentTypes
		}
	}

	pool := &sync.Pool{
		New: func() any {
			w, _ := gzip.NewWriterLevel(io.Discard, cfg.Level)
			return w
		},
	}

	compressible := defaultCompressibleTypes
	if cfg.ContentTypes != nil {
		compressible = make(map[string]bool, len(cfg.ContentTypes))
		for _, ct := range cfg.ContentTypes {
			compressible[ct] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Always set Vary for correct cache behavior.
			w.Header().Add("Vary", "Accept-Encoding")

			// Skip if client doesn't accept gzip.
			if !AcceptsGzip(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Skip WebSocket upgrades.
			if r.Header.Get("Upgrade") != "" {
				next.ServeHTTP(w, r)
				return
			}

			grw := &gzipResponseWriter{
				ResponseWriter: w,
				pool:           pool,
				minSize:        cfg.MinSize,
				compressible:   compressible,
			}
			defer grw.finish()

			next.ServeHTTP(grw, r)
		})
	}
}

type gzipResponseWriter struct {
	http.ResponseWriter
	pool         *sync.Pool
	gzWriter     *gzip.Writer
	minSize      int
	compressible map[string]bool
	buf          []byte // buffer before decision
	statusCode   int
	decided      bool
	compressing  bool
	wroteHeader  bool
}

func (g *gzipResponseWriter) WriteHeader(code int) {
	if g.wroteHeader {
		return
	}
	g.statusCode = code

	// For status codes that have no body, skip compression and write immediately.
	if code == http.StatusNoContent || code == http.StatusNotModified || (code >= 100 && code < 200) {
		g.decided = true
		g.compressing = false
		g.wroteHeader = true
		g.ResponseWriter.WriteHeader(code)
		return
	}

	// If Content-Encoding is already set, skip compression.
	if g.ResponseWriter.Header().Get("Content-Encoding") != "" {
		g.decided = true
		g.compressing = false
		g.wroteHeader = true
		g.ResponseWriter.WriteHeader(code)
		return
	}
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	if !g.wroteHeader {
		g.WriteHeader(http.StatusOK)
	}

	// Already decided: write directly or through gzip.
	if g.decided {
		if g.compressing {
			return g.gzWriter.Write(data)
		}
		return g.ResponseWriter.Write(data)
	}

	// Buffer data until we have enough to decide.
	g.buf = append(g.buf, data...)

	if len(g.buf) >= g.minSize {
		g.decide()
		return len(data), g.flushBuffer()
	}

	return len(data), nil
}

func (g *gzipResponseWriter) decide() {
	g.decided = true

	// Don't compress responses smaller than minSize.
	if len(g.buf) < g.minSize {
		g.compressing = false
		if !g.wroteHeader {
			g.wroteHeader = true
			g.ResponseWriter.WriteHeader(g.statusCode)
		}
		return
	}

	ct := g.ResponseWriter.Header().Get("Content-Type")
	if ct == "" {
		// Sniff content type from buffered data.
		ct = http.DetectContentType(g.buf)
		g.ResponseWriter.Header().Set("Content-Type", ct)
	}

	// Extract media type without parameters.
	mediaType, _, _ := mime.ParseMediaType(ct)

	if !g.compressible[mediaType] {
		g.compressing = false
		if !g.wroteHeader {
			g.wroteHeader = true
			g.ResponseWriter.WriteHeader(g.statusCode)
		}
		return
	}

	// Enable compression.
	g.compressing = true
	g.ResponseWriter.Header().Set("Content-Encoding", "gzip")
	g.ResponseWriter.Header().Del("Content-Length")
	g.wroteHeader = true
	g.ResponseWriter.WriteHeader(g.statusCode)

	g.gzWriter = g.pool.Get().(*gzip.Writer)
	g.gzWriter.Reset(g.ResponseWriter)
}

func (g *gzipResponseWriter) flushBuffer() error {
	if len(g.buf) == 0 {
		return nil
	}
	var err error
	if g.compressing {
		_, err = g.gzWriter.Write(g.buf)
	} else {
		_, err = g.ResponseWriter.Write(g.buf)
	}
	g.buf = nil
	return err
}

func (g *gzipResponseWriter) finish() {
	if !g.decided {
		// Response finished without reaching minSize; decide now.
		g.decide()
		g.flushBuffer()
	}

	if g.compressing && g.gzWriter != nil {
		g.gzWriter.Close()
		g.gzWriter.Reset(io.Discard)
		g.pool.Put(g.gzWriter)
	}
}

// Flush implements http.Flusher.
func (g *gzipResponseWriter) Flush() {
	if !g.decided {
		g.decide()
		g.flushBuffer()
	}

	if g.compressing && g.gzWriter != nil {
		g.gzWriter.Flush()
	}

	if f, ok := g.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack implements http.Hijacker for WebSocket support.
func (g *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := g.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, http.ErrNotSupported
}

// Unwrap returns the underlying ResponseWriter.
func (g *gzipResponseWriter) Unwrap() http.ResponseWriter {
	return g.ResponseWriter
}
