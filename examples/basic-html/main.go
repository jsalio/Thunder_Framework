package main

import (
	"fmt"
	"log"
	"net/http"
	"thunder/internal"

	aboutpage "thunder/examples/basic-html/components/about-page"
	homepage "thunder/examples/basic-html/components/home-page"
	usercard "thunder/examples/basic-html/components/user-card"
)

func main() {
	app := internal.NewApp()

	// ── Estado global (equivalente a signals en el AppComponent de Angular) ──
	app.State.Set("siteName", "Thunder Framework")
	app.State.Set("version", "0.1")

	// ── Archivos estáticos ──
	app.Static("/static/", "./sample/static")

	// ── Registrar componentes ──
	// Cada componente conoce su propia ruta, su template y su handler.
	// main.go solo los orquesta, no contiene lógica de vista.
	homepage.Register(app)
	aboutpage.Register(app)
	usercard.Register(app)

	// ── Partial sin layout (útil para HTMX / fetch parcial) ──
	app.GET("/api/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"%s"}`, app.State.Get("version"))
	})

	fmt.Println(`
  ╔══════════════════════════════════════╗
  ║   Thunder Framework v0.1             ║
  ║   Servidor en http://localhost:8080  ║
  ║   Ctrl+C para detener                ║
  ╚══════════════════════════════════════╝
	`)

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
