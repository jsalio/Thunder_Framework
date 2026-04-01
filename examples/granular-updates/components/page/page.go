package page

import (
	"os"
	"thunder"
	"thunder/component"
)

// Comp defines the main page shell that loads widgets via HTMX.
var Comp = component.Component{
	TemplatePath: componentDir() + "/page.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/page.css",
	Handler: func(ctx *component.Ctx) any {
		return nil
	},
}

// Register adds the main page route.
func Register(app *thunder.App) {
	app.Component("/", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/granular-updates/components/page"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/granular-updates/components/layout"
}
