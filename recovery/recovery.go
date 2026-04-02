// Package recovery provides panic recovery middleware for the Thunder framework.
package recovery

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"thunder/router"
)

// Recover returns a middleware that recovers from panics in downstream handlers.
// It logs the error and stack trace, then responds with HTTP 500.
func Recover() router.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()
					slog.Error("panic recovered",
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
						"stack", string(stack),
					)
					http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
