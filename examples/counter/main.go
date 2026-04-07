package main

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/examples/counter/components"
	"github.com/jsalio/thunder_framework/examples/counter/components/greeting"
)

func main() {
	app := thunder.NewApp()

	// Initial state
	app.State.Set("count", 0)

	// Register global components — usable as <t-greeting /> in any template
	app.RegisterComponent("greeting", greeting.Comp)

	// Register counter page and actions
	components.Register(app)

	app.Run(thunder.AppArgs{
		AppName: "Counter",
		Port:    8090,
	})
}
