package todolist

import (
	"os"
	"thunder/internal"
	"thunder/internal/component"
	todostore "thunder/todo-sample/store"
)

// Comp define el componente TodoList: muestra la lista completa de tareas.
var Comp = component.Component{
	TemplatePath: componentDir() + "/todo-list.html",
	LayoutPath:   layoutDir() + "/layout.html",
	Handler: func(ctx *component.Ctx) any {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		return map[string]any{
			"Todos": ts.All(),
			"Stats": ts.Stats(),
		}
	},
}

// Register registra la ruta GET / de la app TODO.
func Register(app *internal.App) {
	app.Component("/", Comp)
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/todo-sample/components/todo-list"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/todo-sample/components/layout"
}
