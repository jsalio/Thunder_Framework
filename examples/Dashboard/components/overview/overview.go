package overview

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/Dashboard/store"
)

// Comp defines the Dashboard Overview component
var Comp = component.New(func(ctx *component.Ctx) any {
	s := ctx.State.Get("store").(*store.Store)
	return map[string]any{
		"Stats":        s.GetStats(),
		"Transactions": s.GetRecentTransactions(),
	}
}).WithLayout("../layout/layout.html")

// Register adds the overview route
func Register(app *thunder.App) {
	app.Component("/", Comp)
}
