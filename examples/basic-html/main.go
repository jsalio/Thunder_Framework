package main

import (
	"fmt"
	"log"
	"net/http"
	"thunder"

	aboutpage "thunder/examples/basic-html/components/about-page"
	homepage "thunder/examples/basic-html/components/home-page"
	usercard "thunder/examples/basic-html/components/user-card"
)

func main() {
	app := thunder.NewApp()

	// ── Estado global (equivalente a signals en el AppComponent de Angular) ──
	app.State.Set("siteName", "Thunder Framework")
	app.State.Set("version", "0.1")

	// ── Archivos estáticos ──
	app.Static("/static/", "./examples/basic-html/static")

	// ── Registrar componentes ──
	// Each component knows its own route, its template, and its handler.
	// main.go solo los orquesta, no contiene lógica de vista.
	homepage.Register(app)
	aboutpage.Register(app)
	usercard.Register(app)

	// ── Partial sin layout (útil para HTMX / fetch parcial) ──
	app.GET("/api/version", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"version":"%s"}`, app.State.Get("version"))
	})

	if err := app.Run(thunder.AppArgs{
		AppName: "Basic HTML Sample",
		Port:    8080,
	}); err != nil {
		log.Fatal(err)
	}
}
