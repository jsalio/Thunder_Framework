package customers

import (
	"os"
	"thunder"
	"thunder/component"
	"thunder/examples/Dashboard/store"
)

// Comp defines the Customers section component
var Comp = component.Component{
	TemplatePath: componentDir() + "/customers.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/customers.css",
	Handler: func(ctx *component.Ctx) any {
		s := ctx.State.Get("store").(*store.Store)
		summary := s.GetCustomerSummary()
		return map[string]any{
			"Summary":         summary,
			"Customers":       s.GetCustomers(),
			"InactiveChurned": summary.TotalCustomers - summary.ActiveCustomers,
		}
	},
}

// Register adds the customers route
func Register(app *thunder.App) {
	app.Component("/customers", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/Dashboard/components/customers"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/Dashboard/components/layout"
}
