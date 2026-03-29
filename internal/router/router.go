package router

import "net/http"

type Middleware func(http.Handler) http.Handler

type Router struct {
	mux         *http.ServeMux
	middlewares []Middleware
}

func New() *Router {
	return &Router{
		mux:         http.NewServeMux(),
		middlewares: []Middleware{},
	}
}

func (r *Router) Use(m Middleware) {
	r.middlewares = append(r.middlewares, m)
}

func (r *Router) Handle(pattern string, handler http.Handler) {
	r.mux.Handle(pattern, handler)
}

func (r *Router) GET(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("GET "+pattern, handler)
}

func (r *Router) POST(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("POST "+pattern, handler)
}

func (r *Router) PUT(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("PUT "+pattern, handler)
}

func (r *Router) DELETE(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("DELETE "+pattern, handler)
}

func (r *Router) PATCH(pattern string, handler http.HandlerFunc) {
	r.mux.HandleFunc("PATCH "+pattern, handler)
}

func (r *Router) Handler() http.Handler {
	var handler http.Handler = r.mux

	for i := len(r.middlewares) - 1; i >= 0; i-- {
		handler = r.middlewares[i](handler)
	}

	return handler
}
