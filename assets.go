package thunder

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"net/http"

	"github.com/jsalio/thunder_framework/compress"
)

//go:embed assets/js/htmx.min.js
var htmxJS []byte

//go:embed assets/js/idiomorph-ext.min.js
var idiomorphJS []byte

// Pre-compressed versions of embedded assets (computed once at startup).
var (
	htmxJSGzip      []byte
	idiomorphJSGzip []byte
)

func init() {
	htmxJSGzip = precompress(htmxJS)
	idiomorphJSGzip = precompress(idiomorphJS)
}

func precompress(data []byte) []byte {
	var buf bytes.Buffer
	w, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

// registerAssetRoutes registers internal routes to serve the embedded
// HTMX and Idiomorph JavaScript files. These are automatically injected
// into layouts by the preprocessor — no manual <script> tags needed.
func (a *App) registerAssetRoutes() {
	a.Router.Handle("GET /__thunder/htmx.min.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Vary", "Accept-Encoding")
		if compress.AcceptsGzip(r) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(htmxJSGzip)
			return
		}
		w.Write(htmxJS)
	}))

	a.Router.Handle("GET /__thunder/idiomorph-ext.min.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("Vary", "Accept-Encoding")
		if compress.AcceptsGzip(r) {
			w.Header().Set("Content-Encoding", "gzip")
			w.Write(idiomorphJSGzip)
			return
		}
		w.Write(idiomorphJS)
	}))
}
