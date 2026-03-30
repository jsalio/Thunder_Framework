# ⚡ Thunder Framework

**Thunder** is a lightweight, high-performance Go web framework inspired by **Angular's** architecture. It provides a modern develoepr experience with signal-like state management, co-located components, and powerful HTML-native directives, all seamlessly integrated with **HTMX** for building reactive web applications without the complexity of a heavy frontend framework.

---

## ✨ Key Features

- 🏗️ **Angular-Inspired Architecture**: Organized, predictable, and scalable structure.
- 🚦 **Signal-Like State**: Thread-safe global state store for centralized data management.
- 📦 **Co-located Components**: Keep your Go logic, HTML templates, and CSS styles in a single directory.
- 🎨 **HTML-Native Directives**: Use `t-if`, `t-for`, and `t-class` directly in your HTML — no more verbose Go template syntax.
- 🦅 **HTMX Integration Out-of-the-Box**: Built-in support for partial rendering and seamless state updates.
- 🚀 **Gracefully Lightweight**: Fast startup, graceful shutdown, and minimal dependencies.

---

## 🚀 Quick Start

### 1. Installation

```bash
go get github.com/jsalio/thunder_framework
```

### 2. A Simple Component

Create a directory `components/hello` with `hello.go` and `hello.html`:

**hello.go**
```go
package hello

import "thunder/internal/component"

var Comp = component.Component{
    TemplatePath: "hello.html",
    Handler: func(ctx *component.Ctx) any {
        return map[string]any{"Name": "World"}
    },
}
```

**hello.html**
```html
<t-title>Hello Thunder</t-title>
<h1>Hello {{.Name}}!</h1>
```

### 3. Run the App

**main.go**
```go
package main

import (
    "thunder/internal"
    "thunder/examples/hello/components/hello"
)

func main() {
    app := internal.NewApp()
    app.Component("/", hello.Comp)
    app.Run(":8080")
}
```

---

## 🧩 Core Concepts

### 📦 Component System
Thunder encourages co-location. Each feature is a component containing its own logic (`.go`), view (`.html`), and styles (`.css`). Components self-register their routes and actions.

### 🚦 State Management
Manage your application state in a central, thread-safe store. Components can read and mutate state, triggering reactive updates via HTMX.

```go
app.State.Set("count", 0)
// ... in a handler ...
count := ctx.State.Get("count").(int)
ctx.State.Set("count", count + 1)
```

### 🎨 HTML-Native Directives
Stop fighting Go's `{{ if ... }}` syntax. Use attributes instead:

- **Conditionals**: `<div t-if=".IsVisible">...</div>`
- **Loops**: `<li t-for=".Items">...</li>`
- **Dynamic Classes**: `<button t-class-active=".IsActive">Submit</button>`

### ⚡ Actions & HTMX
Register mutations with `app.Action()`. Thunder automatically detects HTMX requests and renders only the necessary component fragment, providing a silky-smooth SPA feel.

```go
app.Action("/todo/add", Comp, func(ctx *component.Ctx) {
    // Logic to add a todo
    // Thunder automatically re-renders the component fragment if it's an HTMX call
})
```

---

## 📂 Project Structure

A typical Thunder project looks like this:

```text
├── components/          # Reusable UI components
│   └── todo-list/
│       ├── todo-list.go
│       ├── todo-list.html
│       └── todo-list.css
├── static/             # Static assets (JS, CSS, Images)
├── templates/          # Global layouts
└── main.go             # Application entry point
```

---

## 🛠️ Requirements

- Go 1.22+
- HTMX (included via CDN in default layouts)

---

## 📄 Documentation

For a deep dive into all features, directives, and APIs, check out the [DOCUMENTATION.md](./DOCUMENTATION.md).

## 🌟 Examples

Check the `examples/` directory for a full **TODO** application showcasing state management, nested components, and HTMX interactions.

---

## 🤝 Contributing

Contributions are welcome! Feel free to open issues or submit PRs to help make Thunder even better.

## 📜 License

Thunder is open-source software licensed under the MIT License.
