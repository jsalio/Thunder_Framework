package main

import (
	"thunder"
	"thunder/examples/hello-world/components/hello"
)

func main() {
	app := thunder.NewApp()
	app.Component("/", hello.Comp)
	app.Run(thunder.AppArgs{
		AppName: "Sample",
		Port:    8086,
	})
}
