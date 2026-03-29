package todolist

import (
	"net/http"
	"os"
	"strconv"
	"thunder/internal"
	"thunder/internal/component"
	todostore "thunder/todo-sample/store"
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
		}
	},
}

// Register registra todas las rutas del componente TodoList.
func Register(app *internal.App) {
	// GET / — renderiza la lista completa
	app.Component("/", Comp)

	// POST /todos — agregar tarea
	app.POST("/todos", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*todostore.TodoStore)
		r.ParseForm()
		text := r.FormValue("text")
		if text != "" {
			ts.Add(text)
		}
		checkSusscess(app, w, r)
	})

	// POST /todos/{id}/done — alternar completado
	app.POST("/todos/{id}/done", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*todostore.TodoStore)
		if id, err := strconv.Atoi(r.PathValue("id")); err == nil {
			ts.Toggle(id)
		}
		checkSusscess(app, w, r)
	})

	// POST /todos/{id}/delete — eliminar tarea
	app.POST("/todos/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*todostore.TodoStore)
		if id, err := strconv.Atoi(r.PathValue("id")); err == nil {
			ts.Delete(id)
		}
		checkSusscess(app, w, r)
	})

	// POST /todos/clear — limpiar completadas
	app.POST("/todos/clear", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*todostore.TodoStore)
		for _, t := range ts.All() {
			if t.Done {
				ts.Delete(t.ID)
			}
		}
		checkSusscess(app, w, r)
	})
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/todo-sample/components/todo-list"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/todo-sample/components/layout"
}

func checkSusscess(app *internal.App, w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		app.RenderComponentPartial(w, r, Comp)
	} else {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}
