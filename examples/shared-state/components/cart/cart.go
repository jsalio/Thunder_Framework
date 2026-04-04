package cart

import (
	"github.com/jsalio/thunder_framework/component"
)

type Item struct {
	Name  string
	Price float64
}

var products = []Item{
	{Name: "Coffee", Price: 3.50},
	{Name: "Sandwich", Price: 7.00},
	{Name: "Cookie", Price: 2.00},
}

// Comp renders the product list and "Add" buttons.
var Comp = component.New(func(ctx *component.Ctx) any {
	raw := ctx.SessionState.Get("cart")
	var items []Item
	if raw != nil {
		items = raw.([]Item)
	}
	return map[string]any{
		"Products": products,
		"Items":    items,
	}
})
