package main

import (
	"log"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/shared-state/components/cart"
	"github.com/jsalio/thunder_framework/examples/shared-state/components/page"
	"github.com/jsalio/thunder_framework/examples/shared-state/components/summary"
)

func main() {
	app := thunder.NewApp()

	// Register global components — usable as <t-cart /> and <t-summary />
	app.RegisterComponent("cart", cart.Comp)
	app.RegisterComponent("summary", summary.Comp)

	// Main page route
	app.Component("/", page.Comp)

	// Action: add a product to the cart.
	// After this, HTMX re-renders only <t-cart /> (the #cart div).
	// The summary sibling is NOT re-rendered — it shows stale data
	// until the user does a full page reload.
	app.Action("/add", cart.Comp, func(ctx *component.Ctx) {
		ctx.Request.ParseForm()
		name := ctx.Request.FormValue("product")

		// Find the product
		var product cart.Item
		for _, p := range []cart.Item{
			{Name: "Coffee", Price: 3.50},
			{Name: "Sandwich", Price: 7.00},
			{Name: "Cookie", Price: 2.00},
		} {
			if p.Name == name {
				product = p
				break
			}
		}

		// Read current cart from SessionState (shared with summary)
		raw := ctx.SessionState.Get("cart")
		var items []cart.Item
		if raw != nil {
			items = raw.([]cart.Item)
		}
		items = append(items, product)

		// Write back — summary can read this on next full render
		ctx.SessionState.Set("cart", items)
	})

	log.Println("Starting Shared State Example...")
	if err := app.Run(thunder.AppArgs{
		AppName: "Shared State",
		Port:    8092,
	}); err != nil {
		log.Fatal(err)
	}
}
