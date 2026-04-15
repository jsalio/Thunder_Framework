package hello

import (
	"github.com/jsalio/thunder_framework/component"
)

var Comp = component.New(func(ctx *component.Ctx) any {
	return map[string]any{"Name": "World"}
})
