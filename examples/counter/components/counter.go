package components

import (
	"os"
	"thunder"
	"thunder/component"
)

var Comp = component.Component{
	TemplatePath: componentDir() + "/counter.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/counter.css",
	Handler: func(ctx *component.Ctx) any {
		count := ctx.SessionState.Get("count")
		if count == nil {
			count = 0
			ctx.SessionState.Set("count", 0)
		}
		return map[string]any{"Count": count}
	},
}

func Register(app *thunder.App) {
	app.Component("/", Comp)

	app.Action("/increment", Comp, func(ctx *component.Ctx) {
		app.Logger.Info("Incrementing session counter")
		current := ctx.SessionState.Get("count")
		val := 0
		if current != nil {
			val = current.(int)
		}
		ctx.SessionState.Set("count", val+1)
	})

	app.Logger.Info("Counter registered")
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/counter/components"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/counter/components/layout"
}
