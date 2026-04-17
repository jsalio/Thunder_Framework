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

import "thunder/component"

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
    "thunder"
    "thunder/examples/hello/components/hello"
)

func main() {
    app := thunder.NewApp()
    app.Component("/", hello.Comp)
    app.Run(thunder.AppArgs{
        AppName: "My App",
        Port:    8080,
    })
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

## 🔥 Live Development with `thunder watch`

Thunder includes a file watcher that rebuilds your Go server and reloads the browser automatically whenever you save a file. No manual restarts, no browser refreshing.

### Installation

Build the `thunder` binary from the `thunder/` directory:

```bash
cd thunder
cargo build --release
# Binary is at thunder/target/release/thunder
# Optionally copy it to your PATH:
cp target/release/thunder /usr/local/bin/thunder
```

Requires [Rust](https://rustup.rs) to build (runtime dependency: none).

### Basic Usage

```bash
thunder watch ./examples/counter
```

This will:

1. Build your Go package
2. Spawn the server
3. Watch for file changes and react automatically
4. Open a WebSocket on port 3001 that triggers browser reloads

### What Happens on Each File Change

| File type | What happens |
| --- | --- |
| `.go` | Server is stopped, package is rebuilt, new server is started, browser reloads |
| `.html` | Browser reloads immediately — no server restart |
| `.css` | Browser reloads immediately — no server restart |
| `.js` | Browser reloads immediately — no server restart |

Asset changes (HTML/CSS/JS) take ~100ms to reach the browser. Go rebuilds take as long as `go build` does — typically under a second with a warm cache.

### Options

```bash
thunder watch <go-package> [options]
```

| Flag | Default | Description |
| --- | --- | --- |
| `<go-package>` | `.` | Path to the Go package to build and run |
| `-d, --watch-dir` | same as package | Directory to watch for changes. Useful if your templates live outside the package directory |
| `-w, --ws-port` | `3001` | WebSocket port the browser connects to for reload signals |
| `--build-first` | off | See below |
| `-e, --extra-ext` | none | Additional file extensions to watch, comma-separated (e.g. `toml,json`) |

### Kill-First vs. Build-First

By default, `thunder watch` uses **kill-first** mode:

1. File saved → old server is stopped immediately
2. `go build` runs (server is down during this time)
3. New server starts → browser reloads

This is safe on all machines and keeps memory usage low. The downside is a brief gap where the server is unavailable during the build.

With `--build-first`, the old server stays alive while the new binary is being compiled:

```bash
thunder watch ./examples/counter --build-first
```

1. File saved → `go build` starts, old server keeps serving
2. Build finishes → old server is stopped, new server starts → browser reloads

This gives zero downtime during development but uses roughly double the memory at peak (two binaries in memory simultaneously). Use it if your build takes long enough that you notice the gap.

### Build Errors

If your Go code has a syntax or compile error, `thunder watch` will print the error to the terminal and **leave the old server running**. A broken save never kills a working server. Fix the error and save again — the watcher picks it up automatically.

### How the Live-Reload Script Gets In

When `thunder watch` spawns your server, it sets two environment variables:

- `THUNDER_WATCHER=1` — tells Thunder to enable live-reload
- `THUNDER_WS_PORT=3001` — tells Thunder which WebSocket port to connect to

Thunder's `Run()` detects these and injects a small WebSocket client script before `</body>` in every layout. The browser connects to the watcher's WebSocket server and calls `location.reload()` on any message. If the watcher exits unexpectedly, the browser retries the connection every 2 seconds.

You do not need to add anything to your templates or Go code for this to work.

### Stopping the Watcher

Press `Ctrl+C`. The watcher sends a graceful shutdown signal to the Go server, waits for it to drain, then exits. No orphan processes, no leaked ports.

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
