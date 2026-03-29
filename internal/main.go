package internal

import (
	"log/slog"
	"net/http"
	"os"
	"thunder/internal/component"
	"thunder/internal/render"
	"thunder/internal/router"
	"thunder/internal/server"
	"thunder/internal/state"
)

type App struct {
	Renderer *render.Engine
	Router   *router.Router
	Logger   *slog.Logger
	State    *state.State
}

func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &App{
		Renderer: render.New("templates", ".html", false),
		Router:   router.New(),
		Logger:   logger,
		State:    state.New(),
	}
}

func NewAppDebug(debug bool) *App {
	return &App{
		Renderer: render.New("templates", ".html", debug),
		State:    state.New(),
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
func (a *App) Static(urlPrefix, dir string) {
	a.Router.Handle(
		"GET "+urlPrefix,
		http.StripPrefix(urlPrefix, http.FileServer(http.Dir(dir))),
	)
}

func (a *App) SetTemplatesDirectory(dir string) {
	a.Renderer.SetDirectory(dir)
}

// Render renderiza una plantilla HTML por nombre (modo legacy).
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

// Component registra un componente en una ruta.
// El componente define su propio template path y handler,
// manteniendo lógica y vista co-locados.
// Ejemplo: app.Component("GET /users/:id", mi_componente.Comp)
func (a *App) Component(pattern string, comp component.Component) {
	a.Router.Handle("GET "+pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &component.Ctx{
			State:   a.State,
			Request: r,
			Params:  extractParams(r),
			Writer:  w,
		}

		var data any
		if comp.Handler != nil {
			data = comp.Handler(ctx)
		}

		// HTMX: renderizar solo el fragmento del componente (sin layout).
		var err error
		if isHTMXRequest(r) {
			err = a.Renderer.RenderPartial(w, comp.TemplatePath, comp.StylePath, data)
		} else {
			err = a.Renderer.RenderFile(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data)
		}
		if err != nil {
			a.Logger.Error("error renderizando componente",
				"template", comp.TemplatePath,
				"error", err,
			)
			http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
		}
	}))
}

// RenderComponent renderiza un componente directamente desde un handler.
// Útil cuando necesitas control adicional sobre la request antes de renderizar.
func (a *App) RenderComponent(w http.ResponseWriter, r *http.Request, comp component.Component) {
	ctx := &component.Ctx{
		State:   a.State,
		Request: r,
		Params:  extractParams(r),
		Writer:  w,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	err := a.Renderer.RenderFile(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data)
	if err != nil {
		a.Logger.Error("error renderizando componente",
			"template", comp.TemplatePath,
			"error", err,
		)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

// isHTMXRequest detecta si la request viene de HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// RenderComponentPartial renderiza un componente sin layout (fragmento HTML).
// Útil para respuestas HTMX desde handlers POST.
func (a *App) RenderComponentPartial(w http.ResponseWriter, r *http.Request, comp component.Component) {
	ctx := &component.Ctx{
		State:   a.State,
		Request: r,
		Params:  extractParams(r),
		Writer:  w,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	err := a.Renderer.RenderPartial(w, comp.TemplatePath, comp.StylePath, data)
	if err != nil {
		a.Logger.Error("error renderizando componente parcial",
			"template", comp.TemplatePath,
			"error", err,
		)
		http.Error(w, "Error interno del servidor", http.StatusInternalServerError)
	}
}

// extractParams extrae los path parameters de la request (Go 1.22+).
func extractParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	// Go 1.22: r.PathValue("param")
	// Extraemos los valores conocidos que puedan estar en la ruta.
	// Los componentes pueden llamar r.PathValue directamente si necesitan.
	_ = r
	return params
}

// Run inicia el servidor HTTP en el puerto indicado.
func (a *App) Run(addr string) error {
	a.Logger.Info("servidor iniciando", "addr", addr)
	return server.Start(addr, a.Router.Handler())
}
