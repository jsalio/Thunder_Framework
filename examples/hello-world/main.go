package main

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/examples/hello-world/components/hello"
)

func main() {
	app := thunder.NewApp()
	app.Component("/", hello.Comp)
	app.Run(thunder.AppArgs{
		AppName: "Sample",
		Port:    8086,
	})
}
