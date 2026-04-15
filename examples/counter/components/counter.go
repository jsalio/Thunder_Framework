package components

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// Comp uses component.New() to auto-detect counter.html and counter.css
// from the same directory as this .go file.
var Comp = component.New(func(ctx *component.Ctx) any {
	count := ctx.SessionState.Get("count")
	if count == nil {
		count = 0
		ctx.SessionState.Set("count", 0)
	}
	return map[string]any{"Count": count}
}).WithLayout("../layout/layout.html")

// Register registers the component and its actions.
func Register(app *thunder.App) {
	app.Component("/", Comp)

	// Action to increment.
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
