package hello

import (
	"os"

	"github.com/jsalio/thunder_framework/component"
)

var Comp = component.Component{
	TemplatePath: componentDir() + "/hello.html",
	StylePath:    componentDir() + "/hello.css",
	Handler: func(ctx *component.Ctx) any {
		return map[string]any{"Name": "World"}
	},
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/hello-world/components/hello"
}
