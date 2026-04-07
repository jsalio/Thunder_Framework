package main

import (
	"log"
	"net/http"
	"os"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
)

// AuthMiddleware is a simple guard that logs access to the admin area.
func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔐 [ACCESS LOG] Access attempt to protected area: %s from %s", r.URL.Path, r.RemoteAddr)
		// In a real app, you would check for a session or JWT here.
		next.ServeHTTP(w, r)
	})
}

func main() {
	app := thunder.NewApp()
	dir, _ := os.Getwd()
	// Relative path to this example's components
	baseDir := dir + "/examples/group-test/components"

	// --- PUBLIC GROUP ---
	// Root group for the public facing website.
	// Uses the blue layout.
	public := app.Group("/")
	
	public.Component("/", component.Component{
		TemplatePath: baseDir + "/public/home.html",
		LayoutPath:   baseDir + "/layout/public_layout.html",
		Handler: func(ctx *component.Ctx) any {
			return map[string]any{"Title": "Inicio"}
		},
	})

	public.Component("/about", component.Component{
		TemplatePath: baseDir + "/public/about.html",
		LayoutPath:   baseDir + "/layout/public_layout.html",
		Handler: func(ctx *component.Ctx) any {
			return map[string]any{"Title": "Sobre Nosotros"}
		},
	})

	// --- ADMIN GROUP ---
	// Group for the management panel.
	// Protected by AuthMiddleware and uses the dark layout.
	admin := app.Group("/admin", AuthMiddleware)

	admin.Component("/dashboard", component.Component{
		TemplatePath: baseDir + "/admin/dashboard.html",
		LayoutPath:   baseDir + "/layout/admin_layout.html",
		Handler: func(ctx *component.Ctx) any {
			return map[string]any{"Title": "Panel de Control"}
		},
	})

	admin.Component("/settings", component.Component{
		TemplatePath: baseDir + "/admin/settings.html",
		LayoutPath:   baseDir + "/layout/admin_layout.html",
		Handler: func(ctx *component.Ctx) any {
			return map[string]any{"Title": "Configuración"}
		},
	})

	// Run the application
	log.Fatal(app.Run(thunder.AppArgs{
		Port:    8088,
		AppName: "Thunder Modular Demo",
	}))
}
