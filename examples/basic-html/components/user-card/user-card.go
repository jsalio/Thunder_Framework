package usercard

import (
	"os"
	"thunder"
	"thunder/component"
)

// Comp defines the UserCard component: route, template, and co-located data.
var Comp = component.Component{
	TemplatePath: componentDir() + "/user-card.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		// r.PathValue() is Go 1.22's native API for path parameters.
		data := ctx.State.Snapshot()
		data["ID"] = ctx.Request.PathValue("id")
		data["Site"] = ctx.State.Get("siteName")
		return data
	},
}

// Register registers the component in the App's router.
func Register(app *thunder.App) {
	app.Component("/users/{id}", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/user-card"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/basic-html/components/layout"
}
