package main

import (
	"fmt"
	"log"
	"thunder/internal"

	todolist "thunder/todo-sample/components/todo-list"
	"thunder/todo-sample/store"
)

func main() {
	app := internal.NewApp()

	// ── Estado global ──────────────────────────────────────────────────────
	app.State.Set("todos", store.New())

	// ── Archivos estáticos ─────────────────────────────────────────────────
	app.Static("/static/", "./todo-sample/static")

	// ── Componentes ────────────────────────────────────────────────────────
	// Cada componente registra sus propias rutas (GET + POST).
	todolist.Register(app)

	fmt.Println(`
  ╔══════════════════════════════════════╗
  ║   Thunder TODO v0.1                  ║
  ║   Servidor en http://localhost:8086  ║
  ║   Ctrl+C para detener                ║
  ╚══════════════════════════════════════╝
	`)

	if err := app.Run(":8086"); err != nil {
		log.Fatal(err)
	}
}
