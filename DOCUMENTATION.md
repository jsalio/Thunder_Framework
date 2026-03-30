# Thunder Framework Documentation

Thunder is a lightweight Go web framework inspired by Angular's architecture, featuring a signal-like state store, co-located components, and a robust template engine.

## Table of Contents
- [State Management](#state-management)
- [Component System](#component-system)
- [Template Engine](#template-engine)
- [Template Directives](#template-directives)
- [Actions (app.Action)](#actions)
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
- `component.Component`: Defines a component with `TemplatePath`, `LayoutPath`, `StylePath`, and `Handler`.

### Component Structure

Each component is a directory with co-located files:

```
components/todo-list/
├── todo-list.go    # Component definition + Register(app)
├── todo-list.html  # Template (uses Thunder directives)
└── todo-list.css   # Scoped styles (optional, injected automatically)
```

### Defining a Component

```go
import "thunder/internal/component"

var Comp = component.Component{
    TemplatePath: componentDir() + "/todo-list.html",
    LayoutPath:   layoutDir() + "/layout.html",   // optional
    StylePath:    componentDir() + "/todo-list.css", // optional
    Handler: func(ctx *component.Ctx) any {
        items := ctx.State.Get("items")
        return map[string]any{"Items": items}
    },
}
```

### Registering a Component

Components self-register all their routes via `Register(app)`:

```go
func Register(app *internal.App) {
    // GET route — renders the component
    app.Component("/", Comp)

    // POST routes — actions that mutate state
    app.Action("/items", Comp, func(ctx *component.Ctx) {
        ctx.Request.ParseForm()
        // ... mutate state ...
    })
}
```

The `main.go` stays clean:

```go
func main() {
    app := internal.NewApp()
    app.State.Set("items", store.New())
    app.Static("/static/", "./static")
    todolist.Register(app)
    app.Run(":8080")
}
```

---

## Template Engine

The `render` package provides a powerful template engine with layout support and fragment rendering.

### Public API
- `render.New(dir, ext, debug)`: Creates a new engine.
- `(*Engine) RenderFile(w, templatePath, layoutPath, stylePath, data)`: Renders a component with layout.
- `(*Engine) RenderPartial(w, templatePath, stylePath, data)`: Renders only the component fragment (HTMX).

### Template Structure

Component templates are plain HTML. The framework auto-wraps the content in `{{define "content"}}...{{end}}` and processes the `<t-title>` tag:

```html
<t-title>Page Title</t-title>

<div class="my-component">
    <h1>Hello {{.Name}}</h1>
</div>
```

- `<t-title>` — Sets the page title (used in the layout's `<title>` tag). The tag is removed from the output.
- Everything else becomes the component's HTML body automatically.

### Layout

The layout wraps the component and provides the HTML shell. No `{{define}}` wrapper is needed — the framework assigns the name automatically:

```html
<!DOCTYPE html>
<html>
<head>
    <title>{{template "title" .}}</title>
    <link rel="stylesheet" href="/static/css/style.css">
    {{block "component-styles" .}}{{end}}
</head>
<body>
    <main>{{template "content" .}}</main>
    <script src="https://unpkg.com/htmx.org@2.0.4"></script>
</body>
</html>
```

- `{{template "content" .}}` — Renders the component's content block.
- `{{block "component-styles" .}}{{end}}` — Automatically injects the component's CSS when `StylePath` is set.

### Value Interpolation

Use `{{.Field}}` to output values from the handler's data:

```html
<span>{{.Name}}</span>
<span>{{index .Stats "Total"}}</span>
```

### Built-in Template Functions

| Function | Description | Example |
|----------|-------------|---------|
| `year` | Current year (int) | `{{year}}` |
| `safeHTML` | Unescaped HTML | `{{safeHTML .Content}}` |
| `safeURL` | Trusted URL | `href="{{safeURL .Link}}"` |
| `safeCSS` | Trusted CSS | `{{safeCSS .Style}}` |
| `safeJS` | Trusted JS | `{{safeJS .Script}}` |
| `safeAttr` | Trusted HTML attribute | `{{safeAttr .Attr}}` |

---

## Template Directives

Thunder provides HTML-native directives that replace Go's verbose template syntax. You write standard HTML attributes; the framework transforms them into Go templates automatically.

### \<t-title\> — Page title

Sets the page title. The `<t-title>` tag is removed from the output and becomes the layout's `<title>` content.

```html
<t-title>My App</t-title>

<div class="content">...</div>
```

If omitted, the layout's `<title>` will be empty. This tag must appear at most once per template.

### t-if — Conditional rendering

Renders the element only when the expression is truthy.

```html
<!-- Show a message when the list is empty -->
<p t-if="not .Items" class="empty">No items yet.</p>

<!-- Expressions with operators (use single quotes to avoid escaping) -->
<div t-if='gt (index .Stats "Done") 0'>
    Has completed items
</div>
```

**Generated output:**
```html
{{if not .Items}}<p class="empty">No items yet.</p>{{end}}
```

### t-else — Else branch

Must appear immediately after a `t-if` element (only whitespace allowed between them).

```html
<p t-if=".Loading">Loading...</p>
<p t-else>Content loaded.</p>
```

**Generated output:**
```html
{{if .Loading}}<p>Loading...</p>{{else}}<p>Content loaded.</p>{{end}}
```

### t-else-if — Else-if branch

Chains conditions after a `t-if`.

```html
<span t-if=".IsAdmin">Admin</span>
<span t-else-if=".IsMod">Moderator</span>
<span t-else>User</span>
```

> **Note:** `t-else` / `t-else-if` must follow their `t-if` sibling with only whitespace between them. Other elements in between will break the pairing.

### t-for — List iteration

Renders the element once for each item in the collection. Inside the loop, `.` refers to the current item.

```html
<li t-for=".Todos" class="todo-item">
    <span>{{.Text}}</span>
</li>
```

**Generated output:**
```html
{{range .Todos}}<li class="todo-item"><span>{{.Text}}</span></li>{{end}}
```

### t-class-NAME — Conditional CSS class

Adds the class `NAME` to the element when the expression is truthy. Can be combined with a static `class` attribute.

```html
<li class="item" t-class-active=".Selected" t-class-disabled=".Locked">
    {{.Label}}
</li>
```

**Generated output:**
```html
<li class="item{{if .Selected}} active{{end}}{{if .Locked}} disabled{{end}}">
    {{.Label}}
</li>
```

### \<template\> — Invisible wrapper

When a directive is placed on a `<template>` element, the `<template>` tags are stripped from the output. Only the inner content is rendered. Useful when you need to wrap multiple sibling elements under a single directive.

```html
<li t-if="not .Todos" class="empty">No items</li>
<template t-else>
    <li t-for=".Todos">{{.Text}}</li>
</template>
```

**Generated output:**
```html
{{if not .Todos}}<li class="empty">No items</li>{{else}}
    {{range .Todos}}<li>{{.Text}}</li>{{end}}
{{end}}
```

Without `<template>`, the `t-else` would need to be on a single element. `<template>` lets you group multiple elements (here, the `t-for` loop) under one directive.

### Combining directives

Directives can be combined on the same element. They process in this order: `t-class-*` first, then `t-for`/`t-if`.

```html
<li t-for=".Todos" class="todo-item" t-class-done=".Done">
    {{.Text}}
</li>
```

**Generated output:**
```html
{{range .Todos}}<li class="todo-item{{if .Done}} done{{end}}">{{.Text}}</li>{{end}}
```

### When to use directives vs raw `{{ }}`

| Scenario | Use | Example |
|----------|-----|---------|
| Show/hide an element | `t-if` | `<div t-if=".Visible">` |
| Loop over a list | `t-for` | `<li t-for=".Items">` |
| Conditional CSS class | `t-class-*` | `t-class-active=".On"` |
| Output a value | `{{.Field}}` | `<span>{{.Name}}</span>` |
| Inline conditional text | `{{if}}...{{end}}` | `{{if .Done}}Yes{{else}}No{{end}}` |
| Conditional attribute value | `{{if}}...{{end}}` | `title="{{if .X}}A{{else}}B{{end}}"` |
| Function calls | `{{func .Arg}}` | `{{index .Map "key"}}` |

**Rule of thumb:** Use `t-*` directives for element-level control flow (wrapping/repeating whole HTML elements). Use `{{ }}` for inline values and expressions within text or attributes.

### Expression syntax

Directive values accept any valid Go template expression:

```html
<!-- Simple field access -->
<div t-if=".Active">

<!-- Negation -->
<div t-if="not .Items">

<!-- Comparison (use single quotes when expression contains double quotes) -->
<div t-if='gt (index .Stats "Done") 0'>

<!-- Function calls -->
<div t-if="eq .Status 1">
```

### Complete example

```html
<t-title>My App</t-title>

<div class="container">
    <ul>
        <li t-if="not .Todos" class="empty">No tasks yet</li>
        <template t-else>
        <li t-for=".Todos" class="task" t-class-completed=".Done">
            <span>{{.Text}}</span>
            <form action="/todos/{{.ID}}/done" method="POST"
                  hx-post="/todos/{{.ID}}/done"
                  hx-target="closest .container"
                  hx-swap="outerHTML">
                <button type="submit">
                    {{if .Done}}Undo{{else}}Complete{{end}}
                </button>
            </form>
        </li>
        </template>
    </ul>
</div>
```

---

## Actions

`app.Action()` registers a POST route linked to a component. The developer writes only the state mutation; the framework handles the response automatically:

- **HTMX request** → re-renders the component as a partial (HTML fragment).
- **Normal request** → redirects back to the referrer (or `/` as fallback).

### Signature

```go
app.Action(pattern string, comp component.Component, handler func(ctx *component.Ctx))
```

### Usage Example

```go
func Register(app *internal.App) {
    app.Component("/", Comp)

    // Add item — only the mutation, no boilerplate
    app.Action("/items", Comp, func(ctx *component.Ctx) {
        store := ctx.State.Get("items").(*ItemStore)
        ctx.Request.ParseForm()
        if text := ctx.Request.FormValue("text"); text != "" {
            store.Add(text)
        }
    })

    // Delete item — path parameters via ctx.Request.PathValue()
    app.Action("/items/{id}/delete", Comp, func(ctx *component.Ctx) {
        store := ctx.State.Get("items").(*ItemStore)
        if id, err := strconv.Atoi(ctx.Request.PathValue("id")); err == nil {
            store.Delete(id)
        }
    })
}
```

### How it works

1. The framework creates a `component.Ctx` with `State`, `Request`, `Params`, and `Writer`.
2. Your handler runs and mutates state.
3. If the request has the `HX-Request: true` header (HTMX), the component is re-rendered as a partial (without layout), and the HTML fragment is sent back.
4. Otherwise, the browser is redirected via HTTP 303 to the `Referer` header (or `/`).

### When to use Action vs POST

| Use `app.Action()` | Use `app.POST()` |
|---------------------|-------------------|
| Standard form mutations (add, edit, delete) | Custom responses (JSON, file download) |
| HTMX-driven partial updates | Redirects to different pages |
| Component re-render after mutation | Non-component handlers |

`app.POST()` is the escape hatch for full control. `app.Action()` is the ergonomic default.

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
