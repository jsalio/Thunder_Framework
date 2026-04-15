package stats

import (
	"math/rand"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// StatsData holds the project statistics stored in session state.
type StatsData struct {
	TasksDone  int
	TasksTotal int
	HoursWeek  float64
	TeamOnline int
	TeamTotal  int
}

func generateStats() StatsData {
	return StatsData{
		TasksDone:  8 + rand.Intn(15),
		TasksTotal: 20 + rand.Intn(10),
		HoursWeek:  25.0 + float64(rand.Intn(20)),
		TeamOnline: 3 + rand.Intn(10),
		TeamTotal:  12,
	}
}

// Comp defines the stats widget (no layout — always renders as fragment).
var Comp = component.New(func(ctx *component.Ctx) any {
		if ctx.SessionState.Get("stats_data") == nil {
			ctx.SessionState.Set("stats_data", generateStats())
			ctx.SessionState.Set("stats_refreshes", 0)
		}
		return map[string]any{
			"Stats":     ctx.SessionState.Get("stats_data").(StatsData),
			"Refreshes": ctx.SessionState.Get("stats_refreshes").(int),
		}
})

// Register adds the stats widget route and its refresh action.
func Register(app *thunder.App) {
	app.Component("/widgets/stats", Comp)
	app.Action("/widgets/stats/refresh", Comp, func(ctx *component.Ctx) {
		ctx.SessionState.Set("stats_data", generateStats())
		refreshes := 0
		if v := ctx.SessionState.Get("stats_refreshes"); v != nil {
			refreshes = v.(int)
		}
		ctx.SessionState.Set("stats_refreshes", refreshes+1)
	})
}
