// Package component defines the structure and context for framework components.
package component

import (
	"net/http"

	"github.com/jsalio/thunder_framework/state"
)

// EventBroadcaster is the interface for broadcasting SSE events.
// Implemented by sse.Hub; used here to avoid circular imports.
type EventBroadcaster interface {
	Broadcast(sessionID, eventName string)
}

// Ctx is the context passed to a component's Handler.
// It's similar to Angular's injection context: accesses global state,
// the request, route parameters, and the response writer.
type Ctx struct {
	State        *state.State
	SessionState *state.State
	Request      *http.Request
	Params       map[string]string
	Writer       http.ResponseWriter
	SessionID    string           // The current session ID
	Broadcaster  EventBroadcaster // SSE event broadcaster (may be nil)
}

// Emit broadcasts a named SSE event to all connections in the current session.
// Sibling or child components listening for this event name will auto-refresh.
func (c *Ctx) Emit(eventName string) {
	if c.Broadcaster != nil && c.SessionID != "" {
		c.Broadcaster.Broadcast(c.SessionID, eventName)
	}
}

// Component unites an HTML template with its data handler.
// It's similar to an Angular @Component: logic and view co-located.
type Component struct {
	// TemplatePath is the path to the component's .html file.
	// It must be relative to the working directory or absolute.
	TemplatePath string

	// LayoutPath is the optional path to an enveloping layout.
	// If empty, the component is rendered without a layout (partial).
	LayoutPath string

	// StylePath is the optional path to the component's co-located .css file.
	// If set, the CSS is injected as an inline <style> tag.
	StylePath string

	// Handler is the function that provides data to the template.
	// It returns any value that will be passed as "data" to the template.
	Handler func(ctx *Ctx) any

	// Children maps logical names to child components for template composition.
	// Use {{child "name"}} in templates to render a child inline.
	Children map[string]Component

	// dir is the auto-detected directory of the component's .go file.
	// Used internally to resolve relative paths in WithLayout, WithStyle, etc.
	dir string
}
