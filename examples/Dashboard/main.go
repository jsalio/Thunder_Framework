package main

import (
	"log"
	"thunder"
	"thunder/examples/Dashboard/components/overview"
	"thunder/examples/Dashboard/store"
)

func main() {
	app := thunder.NewApp()

	// ── Global State ───────────────────────────────────────────────────────
	app.State.Set("store", store.New())

	// ── Static Files ───────────────────────────────────────────────────────
	app.Static("/static/", "./examples/Dashboard/static")

	// ── Components Registration ────────────────────────────────────────────
	overview.Register(app)

	// ── Server Start ───────────────────────────────────────────────────────
	log.Println("Starting Sales Dashboard Example...")
	if err := app.Run(thunder.AppArgs{
		AppName: "Thunder Dash",
		Port:    8090, // Note: using 8090 because 8080/8086 might be in use
	}); err != nil {
		log.Fatal(err)
	}
}
