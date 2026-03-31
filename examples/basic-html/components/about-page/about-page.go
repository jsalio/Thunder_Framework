package aboutpage

import (
	"os"
	"thunder"
	"thunder/component"
)

// Comp defines the AboutPage component: route, template, and co-located data.
var Comp = component.Component{
	TemplatePath: componentDir() + "/about-page.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		return ctx.State.Snapshot()
	},
}

// Register registers the component in the App's router.
func Register(app *thunder.App) {
	app.Component("/about", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/about-page"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/layout"
}
