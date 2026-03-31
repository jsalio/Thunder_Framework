package thunder

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"thunder/component"
	"thunder/render"
	"thunder/router"
	"thunder/server"
	"thunder/state"

	"github.com/charmbracelet/lipgloss"
)

type AppArgs struct {
	Port    int
	AppName string
}

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

// Action registra una acción POST asociada a un componente.
// El handler ejecuta la mutación; el framework responde automáticamente:
//   - HTMX: re-renderiza el componente como parcial
//   - Normal: redirige al referer (o "/" por defecto)
func (a *App) Action(pattern string, comp component.Component, handler func(ctx *component.Ctx)) {
	a.Router.POST(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := &component.Ctx{
			State:   a.State,
			Request: r,
			Params:  extractParams(r),
			Writer:  w,
		}
		handler(ctx)
		if isHTMXRequest(r) {
			a.RenderComponentPartial(w, r, comp)
		} else {
			ref := r.Referer()
			if ref == "" {
				ref = "/"
			}
			http.Redirect(w, r, ref, http.StatusSeeOther)
		}
	})
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

func Ternary[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}

// Run inicia el servidor HTTP en el puerto indicado.
func (a *App) Run(args AppArgs) error {
	a.Logger.Info("servidor iniciando", "addr", args.Port)

	isDefaultName := args.AppName == ""
	isDefaultPort := args.Port == 0

	defaultAppName := Ternary(isDefaultName, "Thunder", args.AppName)
	defaultPort := Ternary(isDefaultPort, 8086, args.Port)

	printBanner(defaultAppName, defaultPort)

	return server.Start(":"+strconv.Itoa(defaultPort), a.Router.Handler())
}

var (
	purple = lipgloss.Color("#7D56F4")
	green  = lipgloss.Color("#04B575")
	amber  = lipgloss.Color("#FFB100")
	white  = lipgloss.Color("#FFFFFF")
)

func printBanner(appName string, port int) {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(purple).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(purple).
		Padding(1, 2)

	infoStyle := lipgloss.NewStyle().
		Foreground(white)

	urlStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#00ADD8")).
		Underline(true)

	appNameRow := lipgloss.JoinHorizontal(lipgloss.Center,
		infoStyle.Render("App Name: "),
		lipgloss.NewStyle().Bold(true).Render(appName),
		" ",
	)

	portRow := lipgloss.JoinHorizontal(lipgloss.Center,
		infoStyle.Render("Port:     "),
		lipgloss.NewStyle().Bold(true).Render(strconv.Itoa(port)),
		" ",
	)

	urlRow := infoStyle.Render("URL:      ") + urlStyle.Render(fmt.Sprintf("http://localhost:%d", port))

	content := lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render("⚡ THUNDER FRAMEWORK 0.1.0 ⚡"),
		"",
		appNameRow,
		portRow,
		urlRow,
		"",
		lipgloss.NewStyle().Italic(true).Faint(true).Render("Press Ctrl+C to stop"),
	)

	fmt.Println(borderStyle.Render(content))
}
