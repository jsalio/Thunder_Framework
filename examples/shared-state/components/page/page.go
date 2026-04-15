package page

import (
	"github.com/jsalio/thunder_framework/component"
)

// Comp is the main page shell. It has no data of its own —
// the sibling components <t-cart /> and <t-summary /> provide their own data.
var Comp = component.New(func(ctx *component.Ctx) any {
	return nil
}).WithLayout("../layout/layout.html")
