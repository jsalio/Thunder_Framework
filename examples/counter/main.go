package main

import (
	"thunder"
	"thunder/examples/counter/components"
)

func main() {
	app := thunder.NewApp()

	// Initial state
	app.State.Set("count", 0)

	// Register counter
	components.Register(app)

	app.Run(thunder.AppArgs{
		AppName: "Counter",
		Port:    8090,
	})
}
