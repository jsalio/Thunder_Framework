package thunder

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"
	"thunder/component"
	"thunder/compress"
	"thunder/recovery"
	"thunder/render"
	"thunder/router"
	"thunder/server"
	"thunder/state"
	"crypto/rand"
	"encoding/hex"

	"github.com/charmbracelet/lipgloss"
)

type AppArgs struct {
	Port               int
	AppName            string
	DisableCompression bool
}

type App struct {
	Renderer *render.Engine
	Router   *router.Router
	Logger   *slog.Logger
	State    *state.State
	Sessions *state.SessionStore
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
		Sessions: state.NewSessionStore(),
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

// POST registers a route for the POST method.
func (a *App) POST(pattern string, handler http.HandlerFunc) {
	a.Router.POST(pattern, handler)
}

// Static serves static files from a directory.
func (a *App) Static(urlPrefix, dir string) {
	a.Router.Handle(
		"GET "+urlPrefix,
		http.StripPrefix(urlPrefix, http.FileServer(http.Dir(dir))),
	)
}

func (a *App) SetTemplatesDirectory(dir string) {
	a.Renderer.SetDirectory(dir)
}

// Render renders an HTML template by name (legacy mode).
func (a *App) Render(w http.ResponseWriter, templateName string, data any) {
	err := a.Renderer.Render(w, templateName, data)
	if err != nil {
		a.Logger.Error("error rendering template",
			"template", templateName,
			"error", err,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// Component registers a component on a route.
// The component defines its own template path and handler,
// keeping logic and view co-located.
// Example: app.Component("GET /users/:id", my_component.Comp)
func (a *App) Component(pattern string, comp component.Component) {
	a.Router.Handle("GET "+pattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := &component.Ctx{
			State:        a.State,
			SessionState: a.getSessionState(w, r),
			Request:      r,
			Params:       extractParams(r),
			Writer:       w,
		}

		var data any
		if comp.Handler != nil {
			data = comp.Handler(ctx)
		}

		// HTMX: render only the component fragment (without layout).
		var err error
		if isHTMXRequest(r) {
			err = a.Renderer.RenderPartial(w, comp.TemplatePath, comp.StylePath, data)
		} else {
			err = a.Renderer.RenderFile(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data)
		}
		if err != nil {
			a.Logger.Error("error rendering component",
				"template", comp.TemplatePath,
				"error", err,
			)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}))
}

// Action registers a POST action associated with a component.
// The handler executes the mutation; the framework responds automatically:
//   - HTMX: re-renders the component as a partial
//   - Normal: redirects to referer (or "/" by default)
func (a *App) Action(pattern string, comp component.Component, handler func(ctx *component.Ctx)) {
	a.Router.POST(pattern, func(w http.ResponseWriter, r *http.Request) {
		ctx := &component.Ctx{
			State:        a.State,
			SessionState: a.getSessionState(w, r),
			Request:      r,
			Params:       extractParams(r),
			Writer:       w,
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

// RenderComponent renders a component directly from a handler.
// Useful when you need extra control over the request before rendering.
func (a *App) RenderComponent(w http.ResponseWriter, r *http.Request, comp component.Component) {
	ctx := &component.Ctx{
		State:        a.State,
		SessionState: a.getSessionState(w, r),
		Request:      r,
		Params:       extractParams(r),
		Writer:       w,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	err := a.Renderer.RenderFile(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data)
	if err != nil {
		a.Logger.Error("error rendering component",
			"template", comp.TemplatePath,
			"error", err,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// isHTMXRequest detects if the request comes from HTMX.
func isHTMXRequest(r *http.Request) bool {
	return r.Header.Get("HX-Request") == "true"
}

// RenderComponentPartial renders a component without layout (HTML fragment).
// Useful for HTMX responses from POST handlers.
func (a *App) RenderComponentPartial(w http.ResponseWriter, r *http.Request, comp component.Component) {
	ctx := &component.Ctx{
		State:        a.State,
		SessionState: a.getSessionState(w, r),
		Request:      r,
		Params:       extractParams(r),
		Writer:       w,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	err := a.Renderer.RenderPartial(w, comp.TemplatePath, comp.StylePath, data)
	if err != nil {
		a.Logger.Error("error rendering partial component",
			"template", comp.TemplatePath,
			"error", err,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// extractParams extracts path parameters from the request (Go 1.22+).
func extractParams(r *http.Request) map[string]string {
	params := make(map[string]string)
	// Go 1.22: r.PathValue("param")
	// We extract known values that may be in the route.
	// Components can call r.PathValue directly if needed.
	_ = r
	return params
}

func (a *App) getSessionState(w http.ResponseWriter, r *http.Request) *state.State {
	cookie, err := r.Cookie("thunder_session")
	var sessionID string
	if err != nil {
		sessionID = generateSessionID()
		http.SetCookie(w, &http.Cookie{
			Name:     "thunder_session",
			Value:    sessionID,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   3600,
		})
	} else {
		sessionID = cookie.Value
	}
	return a.Sessions.Get(sessionID)
}

func generateSessionID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "static-session-id"
	}
	return hex.EncodeToString(b)
}

func Ternary[T any](condition bool, trueVal, falseVal T) T {
	if condition {
		return trueVal
	}
	return falseVal
}

// Run starts the HTTP server on the indicated port.
func (a *App) Run(args AppArgs) error {
	if os.Getenv("THUNDER_WATCHER") == "1" {
		if wsPort, err := strconv.Atoi(os.Getenv("THUNDER_WS_PORT")); err == nil && wsPort > 0 {
			a.Renderer.SetLiveReload(wsPort)
		}
	}

	a.registerAssetRoutes()

	a.Router.Prepend(recovery.Recover())

	if !args.DisableCompression {
		a.Router.Prepend(compress.Gzip())
	}

	a.Logger.Info("server starting", "addr", args.Port)

	// Start background session cleanup every minute
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		for range ticker.C {
			a.Sessions.Cleanup(1 * time.Hour) // 1 hour TTL
		}
	}()

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
