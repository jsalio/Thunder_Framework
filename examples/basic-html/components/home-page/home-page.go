package homepage

import (
	"os"
	"thunder"
	"thunder/component"
)

// Comp define el componente HomePage: ruta, template y datos co-locados.
var Comp = component.Component{
	TemplatePath: componentDir() + "/home-page.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		return ctx.State.Snapshot()
	},
}

// Register registra el componente en el router del App.
func Register(app *internal.App) {
	app.Component("/", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/home-page"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/layout"
}
