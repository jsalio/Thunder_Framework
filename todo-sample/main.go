package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"thunder/internal"

	todolist "thunder/todo-sample/components/todo-list"
	"thunder/todo-sample/store"
)

func main() {
	app := internal.NewApp()

	// ── Estado global (signals) ────────────────────────────────────────────
	app.State.Set("todos", store.New())

	// ── Archivos estáticos ─────────────────────────────────────────────────
	app.Static("/static/", "./todo-sample/static")

	// ── Componentes (GET) ──────────────────────────────────────────────────
	// La lista de TODOs conoce su propia ruta, template y datos.
	todolist.Register(app)

	// ── Acciones (POST) — PRG Pattern ─────────────────────────────────────
	// Las acciones POST redirigen siempre a GET / para evitar resubmit.

	// Agregar tarea
	app.POST("/todos", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*store.TodoStore)
		r.ParseForm()
		text := r.FormValue("text")
		if text != "" {
			ts.Add(text)
		}
		if r.Header.Get("HX-Request") == "true" {
			app.RenderComponentPartial(w, r, todolist.Comp)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	// Alternar completado
	app.POST("/todos/{id}/done", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*store.TodoStore)
		if id, err := strconv.Atoi(r.PathValue("id")); err == nil {
			ts.Toggle(id)
		}
		if r.Header.Get("HX-Request") == "true" {
			app.RenderComponentPartial(w, r, todolist.Comp)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	// Eliminar tarea
	app.POST("/todos/{id}/delete", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*store.TodoStore)
		if id, err := strconv.Atoi(r.PathValue("id")); err == nil {
			ts.Delete(id)
		}
		if r.Header.Get("HX-Request") == "true" {
			app.RenderComponentPartial(w, r, todolist.Comp)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	// Limpiar todas las completadas
	app.POST("/todos/clear", func(w http.ResponseWriter, r *http.Request) {
		ts := app.State.Get("todos").(*store.TodoStore)
		for _, t := range ts.All() {
			if t.Done {
				ts.Delete(t.ID)
			}
		}
		if r.Header.Get("HX-Request") == "true" {
			app.RenderComponentPartial(w, r, todolist.Comp)
		} else {
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}
	})

	fmt.Println(`
  ╔══════════════════════════════════════╗
  ║   Thunder TODO v0.1                  ║
  ║   Servidor en http://localhost:8081  ║
  ║   Ctrl+C para detener                ║
  ╚══════════════════════════════════════╝
	`)

	if err := app.Run(":8086"); err != nil {
		log.Fatal(err)
	}
}
