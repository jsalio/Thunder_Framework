package main

import (
	"log"
	"thunder/internal"

	todolist "thunder/examples/todo-sample/components/todo-list"
	"thunder/examples/todo-sample/store"
)

func main() {
	app := internal.NewApp()

	// ── Estado global ──────────────────────────────────────────────────────
	app.State.Set("todos", store.New())

	// ── Archivos estáticos ─────────────────────────────────────────────────
	app.Static("/static/", "./examples/todo-sample/static")

	todolist.Register(app)

	if err := app.Run(internal.AppArgs{
		AppName: "Todo Sample",
		Port:    8086,
	}); err != nil {
		log.Fatal(err)
	}
}
