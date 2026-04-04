package tasks

import (
	"os"
	"strconv"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// Task represents a single task item.
type Task struct {
	ID    int
	Title string
	Done  bool
}

func getOrInitTasks(ctx *component.Ctx) []Task {
	val := ctx.SessionState.Get("tasks")
	if val == nil {
		defaults := []Task{
			{ID: 1, Title: "Design new landing page", Done: false},
			{ID: 2, Title: "Review PR #42", Done: true},
			{ID: 3, Title: "Update API documentation", Done: false},
			{ID: 4, Title: "Fix navigation bug", Done: false},
			{ID: 5, Title: "Write unit tests for auth", Done: true},
		}
		ctx.SessionState.Set("tasks", defaults)
		ctx.SessionState.Set("tasks_next_id", 6)
		return defaults
	}
	return val.([]Task)
}

// Comp defines the tasks widget (no layout — always renders as fragment).
var Comp = component.Component{
	TemplatePath: componentDir() + "/tasks.html",
	StylePath:    componentDir() + "/tasks.css",
	Handler: func(ctx *component.Ctx) any {
		taskList := getOrInitTasks(ctx)
		pending := 0
		done := 0
		for _, t := range taskList {
			if t.Done {
				done++
			} else {
				pending++
			}
		}
		return map[string]any{
			"Tasks":   taskList,
			"Pending": pending,
			"Done":    done,
		}
	},
}

// Register adds the tasks widget route and its actions.
func Register(app *thunder.App) {
	app.Component("/widgets/tasks", Comp)

	app.Action("/widgets/tasks/add", Comp, func(ctx *component.Ctx) {
		title := ctx.Request.FormValue("title")
		if title == "" {
			return
		}
		taskList := getOrInitTasks(ctx)
		nextID := ctx.SessionState.Get("tasks_next_id").(int)
		taskList = append(taskList, Task{ID: nextID, Title: title, Done: false})
		ctx.SessionState.Set("tasks", taskList)
		ctx.SessionState.Set("tasks_next_id", nextID+1)
	})

	app.Action("/widgets/tasks/toggle", Comp, func(ctx *component.Ctx) {
		idStr := ctx.Request.FormValue("id")
		id, _ := strconv.Atoi(idStr)
		taskList := getOrInitTasks(ctx)
		for i, t := range taskList {
			if t.ID == id {
				taskList[i].Done = !taskList[i].Done
				break
			}
		}
		ctx.SessionState.Set("tasks", taskList)
	})

	app.Action("/widgets/tasks/delete", Comp, func(ctx *component.Ctx) {
		idStr := ctx.Request.FormValue("id")
		id, _ := strconv.Atoi(idStr)
		taskList := getOrInitTasks(ctx)
		for i, t := range taskList {
			if t.ID == id {
				taskList = append(taskList[:i], taskList[i+1:]...)
				break
			}
		}
		ctx.SessionState.Set("tasks", taskList)
	})
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/granular-updates/components/tasks"
}
