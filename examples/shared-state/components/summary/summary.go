package summary

import (
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/shared-state/components/cart"
)

// Comp reads the same SessionState as cart and displays totals.
var Comp = component.New(func(ctx *component.Ctx) any {
	raw := ctx.SessionState.Get("cart")
	var items []cart.Item
	if raw != nil {
		items = raw.([]cart.Item)
	}

	var total float64
	for _, item := range items {
		total += item.Price
	}

	return map[string]any{
		"Count": len(items),
		"Total": total,
	}
})
