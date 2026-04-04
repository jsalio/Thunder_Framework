package activity

import (
	"os"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// Activity represents a team activity entry.
type Activity struct {
	User    string
	Action  string
	Target  string
	Time    string
	Type    string // "comment", "commit", "review"
	Initial string
}

var allActivities = []Activity{
	{User: "Alice Johnson", Action: "commented on", Target: "Issue #128", Time: "2 min ago", Type: "comment", Initial: "A"},
	{User: "Bob Smith", Action: "pushed to", Target: "feature/auth", Time: "15 min ago", Type: "commit", Initial: "B"},
	{User: "Carla Reyes", Action: "approved", Target: "PR #45", Time: "1 hour ago", Type: "review", Initial: "C"},
	{User: "Dave Chen", Action: "commented on", Target: "PR #42", Time: "2 hours ago", Type: "comment", Initial: "D"},
	{User: "Eve Klein", Action: "pushed to", Target: "main", Time: "3 hours ago", Type: "commit", Initial: "E"},
	{User: "Frank Morales", Action: "requested changes on", Target: "PR #41", Time: "4 hours ago", Type: "review", Initial: "F"},
	{User: "Grace Liu", Action: "pushed to", Target: "fix/layout", Time: "5 hours ago", Type: "commit", Initial: "G"},
	{User: "Alice Johnson", Action: "commented on", Target: "Issue #125", Time: "6 hours ago", Type: "comment", Initial: "A"},
}

// Comp defines the activity feed widget (no layout — always renders as fragment).
var Comp = component.Component{
	TemplatePath: componentDir() + "/activity.html",
	StylePath:    componentDir() + "/activity.css",
	Handler: func(ctx *component.Ctx) any {
		// Read filter from query param, fall back to session, default "all"
		filter := ctx.Request.URL.Query().Get("filter")
		if filter == "" {
			if v := ctx.SessionState.Get("activity_filter"); v != nil {
				filter = v.(string)
			} else {
				filter = "all"
			}
		}
		ctx.SessionState.Set("activity_filter", filter)

		var filtered []Activity
		for _, a := range allActivities {
			if filter == "all" || a.Type == filter {
				filtered = append(filtered, a)
			}
		}

		return map[string]any{
			"Activities": filtered,
			"Filter":     filter,
			"Count":      len(filtered),
		}
	},
}

// Register adds the activity widget route.
func Register(app *thunder.App) {
	app.Component("/widgets/activity", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/granular-updates/components/activity"
}
