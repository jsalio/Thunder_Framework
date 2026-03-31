package main

import (
	// "log"

	"thunder/examples/login/components/hello"
	"thunder/internal"
)

func main() {
	app := internal.NewApp()
	app.Component("/", hello.Comp)
	app.Run(internal.AppArgs{
		AppName: "Sample",
		Port:    8086,
	})
}
