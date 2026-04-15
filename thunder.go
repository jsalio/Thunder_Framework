package thunder

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"html/template"

	"github.com/charmbracelet/lipgloss"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/compress"
	"github.com/jsalio/thunder_framework/csrf"
	"github.com/jsalio/thunder_framework/recovery"
	"github.com/jsalio/thunder_framework/render"
	"github.com/jsalio/thunder_framework/router"
	"github.com/jsalio/thunder_framework/server"
	"github.com/jsalio/thunder_framework/sse"
	"github.com/jsalio/thunder_framework/state"
)

type AppArgs struct {
	Port               int
	AppName            string
	DisableCompression bool
	DisableCSRF        bool
	CSRFExempt         []string // paths exempt from CSRF validation
}

type App struct {
	Renderer   *render.Engine
	Router     *router.Router
	Logger     *slog.Logger
	State      *state.State
	Sessions   *state.SessionStore
	SSEHub     *sse.Hub
	components map[string]component.Component
}

func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	return &App{
		Renderer:   render.New("templates", ".html", false),
		Router:     router.New(),
		Logger:     logger,
		State:      state.New(),
		Sessions:   state.NewSessionStore(),
		SSEHub:     sse.NewHub(),
		components: make(map[string]component.Component),
	}
}

func NewAppDebug(debug bool) *App {
	return &App{
		Renderer: render.New("templates", ".html", debug),
		State:    state.New(),
	}
}

// RegisterComponent adds a component to the global registry.
// Registered components can be used in any template as <t-NAME />.
func (a *App) RegisterComponent(name string, comp component.Component) {
	a.components[name] = comp
	a.syncKnownComponents()
}

// syncKnownComponents updates the render engine with the current set of registered names.
func (a *App) syncKnownComponents() {
	known := make(map[string]bool, len(a.components))
	for name := range a.components {
		known[name] = true
	}
	a.Renderer.SetKnownComponents(known)
}

// Components returns the registered component registry (read-only use).
func (a *App) Components() map[string]component.Component {
	return a.components
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
		sessionState, sessionID := a.getSessionStateAndID(w, r)
		ctx := &component.Ctx{
			State:        a.State,
			SessionState: sessionState,
			Request:      r,
			Params:       extractParams(r),
			Writer:       w,
			SessionID:    sessionID,
			Broadcaster:  a.SSEHub,
		}

		var data any
		if comp.Handler != nil {
			data = comp.Handler(ctx)
		}

		token := csrf.Token(r)
		childFuncs := a.buildTemplateFuncs(ctx, comp.Children, token)

		var err error
		if childFuncs != nil {
			// Has children or global components — use RenderWithFuncs to inject {{child}}/{{component}}.
			layoutPath := comp.LayoutPath
			if isHTMXRequest(r) {
				layoutPath = ""
			}
			err = a.Renderer.RenderWithFuncs(w, comp.TemplatePath, layoutPath, comp.StylePath, data, token, childFuncs)
		} else if isHTMXRequest(r) {
			err = a.Renderer.RenderPartialWithCSRF(w, comp.TemplatePath, comp.StylePath, data, token)
		} else {
			err = a.Renderer.RenderFileWithCSRF(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data, token)
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
		sessionState, sessionID := a.getSessionStateAndID(w, r)
		ctx := &component.Ctx{
			State:        a.State,
			SessionState: sessionState,
			Request:      r,
			Params:       extractParams(r),
			Writer:       w,
			SessionID:    sessionID,
			Broadcaster:  a.SSEHub,
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
	sessionState, sessionID := a.getSessionStateAndID(w, r)
	ctx := &component.Ctx{
		State:        a.State,
		SessionState: sessionState,
		Request:      r,
		Params:       extractParams(r),
		Writer:       w,
		SessionID:    sessionID,
		Broadcaster:  a.SSEHub,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	err := a.Renderer.RenderFileWithCSRF(w, comp.TemplatePath, comp.LayoutPath, comp.StylePath, data, csrf.Token(r))
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
	sessionState, sessionID := a.getSessionStateAndID(w, r)
	ctx := &component.Ctx{
		State:        a.State,
		SessionState: sessionState,
		Request:      r,
		Params:       extractParams(r),
		Writer:       w,
		SessionID:    sessionID,
		Broadcaster:  a.SSEHub,
	}

	var data any
	if comp.Handler != nil {
		data = comp.Handler(ctx)
	}

	token := csrf.Token(r)
	childFuncs := a.buildTemplateFuncs(ctx, comp.Children, token)

	var err error
	if childFuncs != nil {
		err = a.Renderer.RenderWithFuncs(w, comp.TemplatePath, "", comp.StylePath, data, token, childFuncs)
	} else {
		err = a.Renderer.RenderPartialWithCSRF(w, comp.TemplatePath, comp.StylePath, data, token)
	}
	if err != nil {
		a.Logger.Error("error rendering partial component",
			"template", comp.TemplatePath,
			"error", err,
		)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// buildTemplateFuncs creates per-request template functions for {{child "name"}}
// and {{component "name"}}. The child function resolves from the component's
// Children map; the component function resolves from the global registry.
func (a *App) buildTemplateFuncs(ctx *component.Ctx, children map[string]component.Component, csrfToken string) template.FuncMap {
	hasChildren := len(children) > 0
	hasGlobalComponents := len(a.components) > 0

	if !hasChildren && !hasGlobalComponents {
		return nil
	}

	funcs := template.FuncMap{}

	if hasChildren {
		funcs["child"] = func(name string) template.HTML {
			child, ok := children[name]
			if !ok {
				a.Logger.Error("child component not found", "name", name)
				return template.HTML("<!-- child " + name + " not found -->")
			}
			var data any
			if child.Handler != nil {
				data = child.Handler(ctx)
			}
			html, err := a.Renderer.RenderPartialToString(child.TemplatePath, child.StylePath, data, csrfToken)
			if err != nil {
				a.Logger.Error("error rendering child component",
					"name", name,
					"template", child.TemplatePath,
					"error", err,
				)
				return template.HTML("<!-- error rendering child " + name + " -->")
			}
			return template.HTML(html)
		}
	}

	if hasGlobalComponents {
		funcs["component"] = func(name string) template.HTML {
			comp, ok := a.components[name]
			if !ok {
				a.Logger.Error("registered component not found", "name", name)
				return template.HTML("<!-- component " + name + " not found -->")
			}
			var data any
			if comp.Handler != nil {
				data = comp.Handler(ctx)
			}
			html, err := a.Renderer.RenderPartialToString(comp.TemplatePath, comp.StylePath, data, csrfToken)
			if err != nil {
				a.Logger.Error("error rendering component",
					"name", name,
					"template", comp.TemplatePath,
					"error", err,
				)
				return template.HTML("<!-- error rendering component " + name + " -->")
			}
			return template.HTML(html)
		}
	}

	return funcs
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

// getSessionStateAndID retrieves the session state and session ID
// from the request cookie. Creates a new session if none exists.
func (a *App) getSessionStateAndID(w http.ResponseWriter, r *http.Request) (*state.State, string) {
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
	return a.Sessions.Get(sessionID), sessionID
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
	a.registerSSERoutes()

	a.Router.Prepend(recovery.Recover())

	if !args.DisableCSRF {
		cfg := csrf.Config{}
		// Always exempt the SSE events endpoint from CSRF.
		exempt := make(map[string]bool)
		exempt["/__thunder/events"] = true
		for _, p := range args.CSRFExempt {
			exempt[p] = true
		}
		cfg.Exempt = exempt
		a.Router.Use(csrf.Protect(cfg))
	}

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

// registerSSERoutes sets up the SSE events endpoint and
// component partial render endpoints for SSE-driven swaps.
func (a *App) registerSSERoutes() {
	// SSE stream endpoint — clients connect here to listen for events.
	a.Router.Handle("GET /__thunder/events", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("thunder_session")
		if err != nil {
			http.Error(w, "no session", http.StatusUnauthorized)
			return
		}
		a.SSEHub.ServeHTTP(w, r, cookie.Value)
	}))

	// Component partial render endpoints — one per registered component.
	// The client JS fetches these to get fresh HTML when an SSE event fires.
	for name, comp := range a.components {
		comp := comp // capture
		a.Router.Handle("GET /__thunder/component/"+name, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			a.RenderComponentPartial(w, r, comp)
		}))
	}
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
