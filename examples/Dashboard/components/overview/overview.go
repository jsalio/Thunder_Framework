package overview

import (
	"os"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/Dashboard/store"
)

// Comp defines the Dashboard Overview component
var Comp = component.Component{
	TemplatePath: componentDir() + "/overview.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/overview.css",
	Handler: func(ctx *component.Ctx) any {
		s := ctx.State.Get("store").(*store.Store)
		return map[string]any{
			"Stats":        s.GetStats(),
			"Transactions": s.GetRecentTransactions(),
		}
	},
}

// Register adds the overview route
func Register(app *thunder.App) {
	app.Component("/", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/Dashboard/components/overview"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/Dashboard/components/layout"
}
