package main

import (
	// "log"

	"thunder/examples/login/components/hello"
	"thunder"
)

func main() {
	app := thunder.NewApp()
	app.Component("/", hello.Comp)
	app.Run(thunder.AppArgs{
		AppName: "Sample",
		Port:    8086,
	})
}
