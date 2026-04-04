// Package component defines the structure and context for framework components.
package component

import (
	"net/http"

	"github.com/jsalio/thunder_framework/form"
	"github.com/jsalio/thunder_framework/state"
)

// Ctx is the context passed to a component's Handler.
// It's similar to Angular's injection context: accesses global state,
// the request, route parameters, and the response writer.
type Ctx struct {
	State        *state.State
	SessionState *state.State
	Request      *http.Request
	Params       map[string]string
	Writer       http.ResponseWriter

	formData any   // cached decoded form data (lazy, set on first FormData call)
	formErr  error // cached decode/validation error
	formDone bool  // whether form decode has been attempted
}

// FormData decodes the request's form data into T on first call and caches it.
// Subsequent calls return the cached result without re-parsing.
//
//	type Login struct {
//	    Email string `form:"email" validate:"required"`
//	}
//	data, err := component.FormData[Login](ctx)
func FormData[T any](ctx *Ctx) (T, error) {
	if ctx.formDone {
		if ctx.formErr != nil {
			var zero T
			return zero, ctx.formErr
		}
		return ctx.formData.(T), nil
	}
	ctx.formDone = true
	data, err := form.Decode[T](ctx.Request)
	if err != nil {
		ctx.formErr = err
		var zero T
		return zero, err
	}
	ctx.formData = data
	return data, nil
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
}
