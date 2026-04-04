package main

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/examples/contact-form/components"
)

func main() {
	app := thunder.NewApp()
	components.Register(app)
	app.Run(thunder.AppArgs{
		Port:    8087,
		AppName: "Contact Form",
	})
}
