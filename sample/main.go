package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"thunder/internal"
)

func main() {
	app := internal.NewApp()

	currentDir, _ := os.Getwd()
	log.Println(currentDir + "/sample/templates")
	app.SetTemplatesDirectory(currentDir + "/sample/templates")

	// app.Renderer.SetDebug(true)

	app.GET("/", func(w http.ResponseWriter, r *http.Request) {
		app.Render(w, "home", nil)
	})

	app.GET("/about", func(w http.ResponseWriter, r *http.Request) {
		app.Render(w, "about", nil)
	})

	app.GET("/users/:id", func(w http.ResponseWriter, r *http.Request) {
		app.Render(w, "user", nil)
	})

	fmt.Println(`
  ╔══════════════════════════════════════╗
  ║   mi-framework v0.1                  ║
  ║   Servidor en http://localhost:8080  ║
  ║   Ctrl+C para detener                ║
  ╚══════════════════════════════════════╝
	`)

	if err := app.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
