# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Run Commands

```bash
# Run examples
go run ./examples/hello-world    # Run TODO sample app (serves on :8086)
go run ./examples/todo-sample

# Run Sales Dashboard example (Premium UI, serves on :8090)
go run ./examples/Dashboard

# Run Counter example
go run ./examples/counter        # Session state example (serves on :8090)
go run ./examples/todo-sample    # Full-featured example (serves on :8086)

# Build all packages
go build ./...

# Run tests
go test ./...
```

Module name is `thunder` (see go.mod). Go 1.24.0. Only external dependency is `lipgloss` (startup banner); the framework itself is built entirely on stdlib.

## Architecture

Thunder is a component-oriented Go web framework with co-located `.go` + `.html` + `.css` files per component. Packages live at the root level (`component/`, `compress/`, `csrf/`, `recovery/`, `render/`, `router/`, `server/`, `state/`). The main orchestrator is `thunder.go`.

### App (`thunder.go`)

`App` is the central struct. Users import `thunder` and use it to register components, routes, and middleware.

```go
app := thunder.NewApp()

app.Component("/", myComp)                    // Register component as GET route
app.Action("/do-thing", myComp, actionFn)     // Register POST action tied to component
app.GET("/api/data", handler)                 // Standard GET route
app.POST("/api/data", handler)                // Standard POST route
app.Static("/static/", "./static")            // Serve static files

app.Run(thunder.AppArgs{Port: 8086, AppName: "MyApp"})
```

Exported fields on `App`: `Renderer`, `Router`, `Logger`, `State`, `Sessions`.

### Subsystems

- **Router** (`router/`) — wraps `http.ServeMux` (Go 1.22+ path patterns with `PathValue()`). Supports `GET`, `POST`, `PUT`, `DELETE`, `PATCH`. Global middleware via `app.Router.Use(middleware)`, applied in registration order.
- **Renderer** (`render/`) — Go `html/template` engine with thread-safe caching (disabled in debug mode). Includes a **template preprocessor** that transforms Thunder directives into Go template syntax before parsing. Handles CSS injection and layout composition.
- **State** (`state/`) — thread-safe `sync.RWMutex`-based key-value store. Two scopes: **global** (`app.State`, shared across all users) and **session** (per-user, managed via `SessionStore` with cookie-based session IDs).
- **Server** (`server/`) — HTTP server with graceful shutdown (SIGTERM/SIGINT), 15s read/write timeouts, 60s idle timeout.
- **CSRF** (`csrf/`) — Double-Submit Cookie CSRF protection. Middleware generates token in `thunder_csrf` cookie, validates on POST/PUT/DELETE/PATCH via `X-CSRF-Token` header or `_csrf` form field. Auto-injected into forms by the preprocessor and into HTMX requests via `htmx:configRequest`. Enabled by default; disable with `AppArgs.DisableCSRF`.
- **Recovery** (`recovery/`) — Panic recovery middleware, logs stack traces and returns 500.
- **Compress** (`compress/`) — Gzip compression middleware for eligible responses.

## Component System

### Defining a Component

A component is a `component.Component` struct with co-located files:

```
components/home-page/
├── home-page.go    # Component definition, handler, Register(app) function
├── home-page.html  # Template using {{define "content"}}...{{end}}
└── home-page.css   # Optional component-specific styles

Components self-register via `Register(app)` which calls `app.Component(pattern, comp)`. The handler receives a `component.Ctx` with State, Request, Params, and Writer.

```go
var Comp = component.Component{
    TemplatePath: componentDir() + "/my-page.html",
    LayoutPath:   layoutDir() + "/layout.html",   // optional — omit for partials
    StylePath:    componentDir() + "/my-page.css", // optional
    Handler: func(ctx *component.Ctx) any {
        return map[string]any{"Name": "World"}
    },
}
```

### Component Context (`component.Ctx`)

The handler receives a `Ctx` with:

- `ctx.State` — global app state (`*state.State`)
- `ctx.SessionState` — per-user session state (`*state.State`)
- `ctx.Request` — the `*http.Request` (use `ctx.Request.PathValue("id")` for route params)
- `ctx.Writer` — the `http.ResponseWriter`
- `ctx.Params` — `map[string]string` (currently unused; use `PathValue()` instead)

State methods: `.Get(key) any`, `.Set(key, val)`, `.Snapshot() map[string]any`.

### Registering Components

Components self-register via a `Register(app *thunder.App)` function:

```go
func Register(app *thunder.App) {
    app.Component("/", Comp)                              // GET — renders component
    app.Action("/increment", Comp, func(ctx *component.Ctx) {
        // mutate state, then component re-renders automatically
        count := ctx.SessionState.Get("count").(int)
        ctx.SessionState.Set("count", count+1)
    })
}
```

- `app.Component(pattern, comp)` — registers a GET route. Detects HTMX requests (`HX-Request` header) and renders without layout (partial) for those.
- `app.Action(pattern, comp, handler)` — registers a POST route. Runs the handler, then: HTMX request → re-renders component as partial; normal request → redirects to `Referer` via HTTP 303.

### Additional Render Methods

- `app.RenderComponent(w, r, comp)` — render a component with full layout from a custom handler.
- `app.RenderComponentPartial(w, r, comp)` — render without layout (HTML fragment).
- `app.Render(w, templateName, data)` — legacy rendering from `templates/` directory.

## Template System

### Preprocessor Directives

Thunder preprocesses HTML templates before Go's `html/template` parses them. Page templates are **auto-wrapped** in `{{define "content"}}...{{end}}` — you don't need to write that manually.

| Directive | Example | Description |
|---|---|---|
| `t-if` | `<div t-if=".Active">...</div>` | Conditional rendering |
| `t-else` | `<div t-else>...</div>` | Else branch (must follow `t-if`) |
| `t-else-if` | `<div t-else-if=".Other">...</div>` | Else-if branch |
| `t-for` | `<li t-for=".Items">{{.Name}}</li>` | Loop over collection |
| `t-class-NAME` | `<div t-class-active=".IsActive">` | Conditionally add CSS class |
| `t-morph` | `<div t-morph>...</div>` | Adds `hx-ext="morph"` and `hx-swap="morph:innerHTML"` (Idiomorph) |
| `<t-title>` | `<t-title>My Page</t-title>` | Sets the page title block |
| `<template>` | `<template t-if=".X">...</template>` | Invisible wrapper (stripped from output) |

Use single quotes for expressions containing double quotes: `t-if='gt (index .Stats "Done") 0'`.

### Layout Pattern

Layouts use Go's template block system. The layout calls `{{template "content" .}}` and optionally `{{template "title" .}}` and `{{block "component-styles" .}}{{end}}`:

```html
<!-- layout.html -->
<!DOCTYPE html>
<html>
<head>
    <title>{{template "title" .}}</title>
    {{block "component-styles" .}}{{end}}
</head>
<body>
    {{template "content" .}}
</body>
</html>
```

Component CSS (`StylePath`) is automatically injected into the `component-styles` block.

### Built-in Template Functions

`safeHTML`, `safeJS`, `safeCSS`, `safeURL`, `safeAttr`, `year`. Custom functions via `app.Renderer.AddFunc(name, fn)`.

## HTMX Integration

Thunder has first-class HTMX support. HTMX (`htmx.min.js`) and Idiomorph (`idiomorph-ext.min.js`) are **embedded in the framework binary** and auto-injected into layouts before `</body>` — no manual `<script>` tags or CDN dependencies needed. The files are served from `/__thunder/` routes with immutable caching headers.

When a request includes the `HX-Request: true` header:
- `app.Component()` renders the component **without layout** (partial HTML fragment).
- `app.Action()` re-renders the component as a partial after executing the mutation handler.

This enables seamless partial page updates without full page reloads.

## Session Management

Sessions are cookie-based (`thunder_session` cookie). Session IDs are cryptographically random (16 bytes, hex-encoded). Cookies are `HttpOnly`, `SameSite=Lax`, with 1-hour `MaxAge`. Expired sessions are cleaned up automatically in the background. Hard limit of 5,000 concurrent sessions.

## Work traking

Use 'bd' for task tracking


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
