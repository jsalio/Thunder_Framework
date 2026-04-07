package greeting

import (
	"time"

	"github.com/jsalio/thunder_framework/component"
)

// Comp is a simple greeting component that displays the current time.
var Comp = component.New(func(ctx *component.Ctx) any {
	return map[string]any{
		"Time": time.Now().Format("15:04:05"),
	}
})
