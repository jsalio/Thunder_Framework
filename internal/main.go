package internal

import (
	"log/slog"
	"net/http"
	"os"
	"thunder/internal/render"
	"thunder/internal/router"
	"thunder/internal/server"
)

type App struct {
	Renderer *render.Engine
	Router   *router.Router
	Logger   *slog.Logger
}

func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &App{
		Renderer: render.New("templates", ".html", false),
		Router:   router.New(),
		Logger:   logger,
	}
}

func NewAppDebug(debug bool) *App {
	return &App{
		Renderer: render.New("templates", ".html", debug),
	}
}

func (a *App) GET(pattern string, handler http.HandlerFunc) {
	a.Router.GET(pattern, handler)
}

// POST registra una ruta para el método POST.
func (a *App) POST(pattern string, handler http.HandlerFunc) {
	a.Router.POST(pattern, handler)
}

// Static sirve archivos estáticos desde un directorio.
// Ejemplo: a.Static("/static/", "./static")
func (a *App) Static(urlPrefix, dir string) {
	a.Router.Handle(
		"GET "+urlPrefix,
		http.StripPrefix(urlPrefix, http.FileServer(http.Dir(dir))),
	)
}

func (a *App) SetTemplatesDirectory(dir string) {
	a.Renderer.SetDirectory(dir)
}

// Render renderiza una plantilla HTML y la escribe en el ResponseWriter.
// data puede ser nil si la plantilla no necesita datos.
func (a *App) Render(w http.ResponseWriter, templateName string, data any) {
	err := a.Renderer.Render(w, templateName, data)
	if err != nil {
		a.Logger.Error("error renderizando plantilla",
			"template", templateName,
			"error", err,
		)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

// Run inicia el servidor HTTP en el puerto indicado.
// Ejemplo: app.Run(":8080")
func (a *App) Run(addr string) error {
	a.Logger.Info("servidor iniciando", "addr", addr)
	return server.Start(addr, a.Router.Handler())
}
