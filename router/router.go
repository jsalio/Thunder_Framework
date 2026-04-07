// Package router provides a lightweight HTTP router with middleware support.
package router

import (
	"net/http"
	"strings"
)

// Middleware is a function that wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Router is a wrapper around http.ServeMux that supports middleware.
type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

// Group represents a route group with a common prefix and middleware stack.
type Group struct {
	router      *Router
	prefix      string
	middlewares []Middleware
}

// wrap applies the group's middleware stack to the given handler.
func (g *Group) wrap(handler http.Handler) http.Handler {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return handler
}

// Handle registers a handler for the given pattern within the group.
func (g *Group) Handle(pattern string, handler http.Handler) {
	// Go 1.22 patterns can be "GET /path" or just "/path".
	// We need to inject the prefix correctly.
	method, path, found := strings.Cut(pattern, " ")
	if !found {
		// Pattern is just the path, e.g., "/hello"
		g.router.mux.Handle(g.prefix+pattern, g.wrap(handler))
		return
	}
	// Pattern includes a method, e.g., "GET /hello"
	// Reconstruct as "GET " + prefix + "/hello"
	fullPath := g.prefix + path
	if strings.HasSuffix(g.prefix, "/") && strings.HasPrefix(path, "/") {
		fullPath = g.prefix + path[1:]
	}
	g.router.mux.Handle(method+" "+fullPath, g.wrap(handler))
}

// GET registers a GET handler for the given pattern within the group.
func (g *Group) GET(pattern string, handler http.HandlerFunc) {
	g.Handle("GET "+pattern, handler)
}

// POST registers a POST handler for the given pattern within the group.
func (g *Group) POST(pattern string, handler http.HandlerFunc) {
	g.Handle("POST "+pattern, handler)
}

// PUT registers a PUT handler for the given pattern within the group.
func (g *Group) PUT(pattern string, handler http.HandlerFunc) {
	g.Handle("PUT "+pattern, handler)
}

// DELETE registers a DELETE handler for the given pattern within the group.
func (g *Group) DELETE(pattern string, handler http.HandlerFunc) {
	g.Handle("DELETE "+pattern, handler)
}

// PATCH registers a PATCH handler for the given pattern within the group.
func (g *Group) PATCH(pattern string, handler http.HandlerFunc) {
	g.Handle("PATCH "+pattern, handler)
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

// Group creates a new route group with the given prefix and optional middlewares.
func (r *Router) Group(prefix string, middlewares ...Middleware) *Group {
	return &Group{
		router:      r,
		prefix:      prefix,
		middlewares: middlewares,
	}
}

// Group permits nesting groups.
func (g *Group) Group(prefix string, middlewares ...Middleware) *Group {
	combinedMiddlewares := append(g.middlewares, middlewares...)
	return &Group{
		router:      g.router,
		prefix:      g.prefix + prefix,
		middlewares: combinedMiddlewares,
	}
}
