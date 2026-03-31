package main

import (
	"log"
	"thunder"

	todolist "thunder/examples/todo-sample/components/todo-list"
	"thunder/examples/todo-sample/store"
)

func main() {
	app := thunder.NewApp()

	// ── Estado global ──────────────────────────────────────────────────────
	app.State.Set("todos", store.New())

	// ── Archivos estáticos ─────────────────────────────────────────────────
	app.Static("/static/", "./examples/todo-sample/static")

	todolist.Register(app)

	if err := app.Run(thunder.AppArgs{
		AppName: "Todo Sample",
		Port:    8086,
	}); err != nil {
		log.Fatal(err)
	}
}
