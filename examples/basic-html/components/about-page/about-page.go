package aboutpage

import (
	"os"
	"thunder/internal"
	"thunder/internal/component"
)

// Comp define el componente AboutPage: ruta, template y datos co-locados.
var Comp = component.Component{
	TemplatePath: componentDir() + "/about-page.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		return ctx.State.Snapshot()
	},
}

// Register registra el componente en el router del App.
func Register(app *internal.App) {
	app.Component("/about", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/sample/components/about-page"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/sample/components/layout"
}
