package page

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// Comp defines the main page shell that loads widgets via HTMX.
var Comp = component.New(func(ctx *component.Ctx) any {
	return nil
}).WithLayout("../layout/layout.html")

// Register adds the main page route.
func Register(app *thunder.App) {
	app.Component("/", Comp)
}
