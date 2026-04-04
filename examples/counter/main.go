package main

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/examples/counter/components"
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
