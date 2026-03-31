package todolist

import (
	"os"
	"strconv"
	"thunder"
	"thunder/component"
	todostore "thunder/examples/todo-sample/store"
)

// Comp define el componente TodoList: muestra la lista completa de tareas.
var Comp = component.Component{
	TemplatePath: componentDir() + "/todo-list.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/todo-list.css",
	Handler: func(ctx *component.Ctx) any {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		return map[string]any{
			"Todos": ts.All(),
			"Stats": ts.Stats(),
			"name":  "Jorge",
		}
	},
}

// Register registra todas las rutas del componente TodoList.
func Register(app *thunder.App) {
	app.Component("/", Comp)

	// Agregar tarea
	app.Action("/todos", Comp, func(ctx *component.Ctx) {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		ctx.Request.ParseForm()
		if text := ctx.Request.FormValue("text"); text != "" {
			ts.Add(text)
		}
	})

	// Alternar completado
	app.Action("/todos/{id}/done", Comp, func(ctx *component.Ctx) {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		if id, err := strconv.Atoi(ctx.Request.PathValue("id")); err == nil {
			ts.Toggle(id)
		}
	})

	// Eliminar tarea
	app.Action("/todos/{id}/delete", Comp, func(ctx *component.Ctx) {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		if id, err := strconv.Atoi(ctx.Request.PathValue("id")); err == nil {
			ts.Delete(id)
		}
	})

	// Limpiar completadas
	app.Action("/todos/clear", Comp, func(ctx *component.Ctx) {
		ts := ctx.State.Get("todos").(*todostore.TodoStore)
		for _, t := range ts.All() {
			if t.Done {
				ts.Delete(t.ID)
			}
		}
	})
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/todo-sample/components/todo-list"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/todo-sample/components/layout"
}
