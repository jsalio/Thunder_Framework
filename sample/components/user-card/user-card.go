package usercard

import (
	"os"
	"thunder/internal"
	"thunder/internal/component"
)

// Comp define el componente UserCard: ruta, template y datos co-locados.
var Comp = component.Component{
	TemplatePath: componentDir() + "/user-card.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		// r.PathValue() es la API nativa de Go 1.22 para parámetros de ruta.
		data := ctx.State.Snapshot()
		data["ID"] = ctx.Request.PathValue("id")
		data["Site"] = ctx.State.Get("siteName")
		return data
	},
}

// Register registra el componente en el router del App.
func Register(app *internal.App) {
	app.Component("/users/{id}", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/sample/components/user-card"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/sample/components/layout"
}
