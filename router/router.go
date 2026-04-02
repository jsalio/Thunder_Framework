// Package router provides a lightweight HTTP router with middleware support.
package router

import "net/http"

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Router is a wrapper around http.ServeMux that supports middleware.
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// New creates and initializes a new Router.
func New() *Router {
	return &Router{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

// Use adds a middleware to the router's middleware stack.
func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

// Prepend adds a middleware to the front of the stack (outermost execution).
func (r *Router) Prepend(m Middleware) {
	r.middlewares = append([]Middleware{m}, r.middlewares...)
}

// Handle registers a handler for the given pattern.
func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

// GET registers a GET handler for the given pattern.
func (r *Router) GET(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("GET "+pattern, handler)
}

// POST registers a POST handler for the given pattern.
func (r *Router) POST(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("POST "+pattern, handler)
}

// PUT registers a PUT handler for the given pattern.
func (r *Router) PUT(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("PUT "+pattern, handler)
}

// DELETE registers a DELETE handler for the given pattern.
func (r *Router) DELETE(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("DELETE "+pattern, handler)
}

// PATCH registers a PATCH handler for the given pattern.
func (r *Router) PATCH(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("PATCH "+pattern, handler)
}

// Handler returns the final http.Handler with all middlewares applied.
func (r *Router) Handler() http.Handler {
	var handler http.Handler = r.mux

	// Apply middlewares in reverse order so they execute in the order they were added.
	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	return handler
}
