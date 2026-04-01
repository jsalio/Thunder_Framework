package main

import (
	"log"
	"thunder"
	"thunder/examples/granular-updates/components/activity"
	"thunder/examples/granular-updates/components/page"
	"thunder/examples/granular-updates/components/stats"
	"thunder/examples/granular-updates/components/tasks"
)

func main() {
	app := thunder.NewApp()

	// ── Static Files ───────────────────────────────────────────────────────
	app.Static("/static/", "./examples/granular-updates/static")

	// ── Components Registration ────────────────────────────────────────────
	// Page shell (full layout)
	page.Register(app)

	// Independent widgets (no layout — render as fragments)
	stats.Register(app)
	tasks.Register(app)
	activity.Register(app)

	// ── Server Start ───────────────────────────────────────────────────────
	log.Println("Starting Granular Updates Example...")
	if err := app.Run(thunder.AppArgs{
		AppName: "Granular Updates",
		Port:    8091,
	}); err != nil {
		log.Fatal(err)
	}
}
