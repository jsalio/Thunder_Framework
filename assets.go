package thunder

import (
	_ "embed"
	"net/http"
)

//go:embed assets/js/htmx.min.js
var htmxJS []byte

//go:embed assets/js/idiomorph-ext.min.js
var idiomorphJS []byte

// registerAssetRoutes registers internal routes to serve the embedded
// HTMX and Idiomorph JavaScript files. These are automatically injected
// into layouts by the preprocessor — no manual <script> tags needed.
func (a *App) registerAssetRoutes() {
	a.Router.Handle("GET /__thunder/htmx.min.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(htmxJS)
	}))

	a.Router.Handle("GET /__thunder/idiomorph-ext.min.js", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(idiomorphJS)
	}))
}
