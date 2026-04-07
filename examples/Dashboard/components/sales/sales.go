package sales

import (
	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/examples/Dashboard/store"
)

// ChartBar holds pre-computed data for a bar in the chart.
type ChartBar struct {
	Month   string
	Amount  float64
	Percent float64
}

// Comp defines the Sales section component
var Comp = component.New(func(ctx *component.Ctx) any {
	s := ctx.State.Get("store").(*store.Store)
	summary := s.GetSalesSummary()

	// Compute bar heights as percentages relative to max
	var maxAmount float64
	for _, m := range summary.MonthlySales {
		if m.Amount > maxAmount {
			maxAmount = m.Amount
		}
	}
	bars := make([]ChartBar, len(summary.MonthlySales))
	for i, m := range summary.MonthlySales {
		pct := 0.0
		if maxAmount > 0 {
			pct = (m.Amount / maxAmount) * 100
		}
		bars[i] = ChartBar{Month: m.Month, Amount: m.Amount, Percent: pct}
	}

	return map[string]any{
		"Summary":   summary,
		"Records":   s.GetSalesRecords(),
		"ChartBars": bars,
	}
}).WithLayout("../layout/layout.html")

// Register adds the sales route
func Register(app *thunder.App) {
	app.Component("/sales", Comp)
}
