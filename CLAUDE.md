# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Run sample app (serves on :8080)
go run ./sample

# Run TODO sample app (serves on :8086)
go run ./todo-sample

# Build
go build ./...

# No tests exist yet — standard Go testing applies
go test ./...
```

Module name is `thunder` (see go.mod). Go 1.23.9, zero external dependencies — built entirely on stdlib.

## Architecture

Thunder is a component-oriented Go web framework inspired by Angular's co-location pattern. The framework lives in `internal/` and sample apps demonstrate usage.

### Core Flow

`App` (internal/main.go) is the central orchestrator holding four subsystems:
- **Router** (`internal/router/`) — wraps `http.ServeMux` (Go 1.22+ path patterns with `PathValue()`), supports middleware chain applied in reverse registration order
- **Renderer** (`internal/render/`) — Go `html/template` engine with thread-safe caching (disabled in debug mode), supports layouts and partials
- **State** (`internal/state/`) — thread-safe `sync.RWMutex`-based key-value store for global app state, accessible in component handlers via `ctx.State`
- **Server** (`internal/server/`) — HTTP server with graceful shutdown (SIGTERM/SIGINT), configurable timeouts

### Component System

The key abstraction is `component.Component` (`internal/component/`):

```go
type Component struct {
    TemplatePath string
    LayoutPath   string
    Handler      func(ctx *Ctx) any  // returns template data
}
```

Each component is a directory with co-located `.go` + `.html` files:
```
components/home-page/
├── home-page.go    # Component definition, handler, Register(app) function
└── home-page.html  # Template using {{define "content"}}...{{end}}
```

Components self-register via `Register(app)` which calls `app.Component(pattern, comp)`. The handler receives a `component.Ctx` with State, Request, Params, and Writer.

### Template Pattern

Layouts use Go's `{{template "content" .}}` / `{{define "content"}}` block system. The renderer parses both layout and component template together, executing the layout which delegates to named blocks.

### Routing

`app.GET`, `app.POST`, etc. register standard `http.HandlerFunc` routes. `app.Component(pattern, comp)` registers a component as a GET route. `app.Static(prefix, dir)` serves static files.

### Internal comments are in Spanish.
