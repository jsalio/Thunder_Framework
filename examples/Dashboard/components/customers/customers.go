package customers

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/Dashboard/store"
)

// Comp defines the Customers section component
var Comp = component.New(func(ctx *component.Ctx) any {
	s := ctx.State.Get("store").(*store.Store)
	summary := s.GetCustomerSummary()
	return map[string]any{
		"Summary":         summary,
		"Customers":       s.GetCustomers(),
		"InactiveChurned": summary.TotalCustomers - summary.ActiveCustomers,
	}
}).WithLayout("../layout/layout.html")

// Register adds the customers route
func Register(app *thunder.App) {
	app.Component("/customers", Comp)
}
