# Thunder Framework Documentation

Thunder is a lightweight Go web framework inspired by Angular's architecture, featuring a signal-like state store, co-located components, and a robust template engine.

## Table of Contents
- [State Management](#state-management)
- [Component System](#component-system)
- [Template Engine](#template-engine)
- [Routing & Middleware](#routing--middleware)
- [Server Lifecycle](#server-lifecycle)

---

## State Management

Thunder provides a thread-safe global state store using the `state` package. It works as a central repository for application data.

### Public API
- `state.New()`: Creates a new state store.
- `(*State) Set(key string, val any)`: Stores a value.
- `(*State) Get(key string) any`: Retrieves a value.
- `(*State) Snapshot() map[string]any`: Returns a copy of the entire state.

### Usage Example
```go
import "thunder/internal/state"

s := state.New()
s.Set("user_count", 42)

count := s.Get("user_count").(int)
fmt.Println("User count:", count)
```

---

## Component System

The `component` package defines the structure for co-located logic and views.

### Public Structures
- `component.Ctx`: The context passed to a component handler. Includes `State`, `Request`, `Params`, and `Writer`.
- `component.Component`: Defines a component with a `TemplatePath`, an optional `LayoutPath`, and a `Handler` function.

### Usage Example
```go
import "thunder/internal/component"

myComponent := &component.Component{
    TemplatePath: "ui/button.html",
    Handler: func(ctx *component.Ctx) any {
        // Access global state or request params
        name := ctx.State.Get("app_name")
        return map[string]any{"Name": name}
    },
}
```

---

## Template Engine

The `render` package provides a powerful template engine with layout support and fragment rendering.

### Public API
- `render.New(dir, ext, debug)`: Creates a new engine.
- `(*Engine) RenderFile(w, templatePath, layoutPath, data)`: Renders a component.
- `(*Engine) RenderPartial(w, templatePath, data)`: Renders only the component fragment.

### Usage Example
```go
import "thunder/internal/render"

engine := render.New("./views", ".html", true)

// Render a component with a layout
engine.RenderFile(w, "pages/home.html", "layout/main.html", data)

// Render just a fragment (HTMX style)
engine.RenderPartial(w, "components/sidebar.html", data)
```

---

## Routing & Middleware

The `router` package is a lightweight wrapper around `http.ServeMux` with middleware support.

### Public API
- `router.New()`: Creates a new router.
- `(*Router) Use(Middleware)`: Adds middleware.
- `(*Router) GET/POST/PUT/DELETE/PATCH(pattern, handler)`: Registers routes.
- `(*Router) Handler()`: Returns the final `http.Handler`.

### Usage Example
```go
import "thunder/internal/router"

r := router.New()

// Middleware
r.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
        log.Println(req.URL.Path)
        next.ServeHTTP(w, req)
    })
})

r.GET("/", func(w http.ResponseWriter, req *http.Request) {
    w.Write([]byte("Hello Thunder!"))
})
```

---

## Server Lifecycle

The `server` package handles graceful startup and shutdown.

### Public API
- `server.Start(addr, handler)`: Starts the server and waits for interruption signals (SIGINT/SIGTERM).

### Usage Example
```go
import (
    "thunder/internal/router"
    "thunder/internal/server"
)

r := router.New()
// ... register routes ...

err := server.Start(":8080", r.Handler())
if err != nil {
    log.Fatal(err)
}
```
