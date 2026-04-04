package components

import (
	"os"

	thunder "github.com/jsalio/thunder_framework"
	"github.com/jsalio/thunder_framework/component"
	"github.com/jsalio/thunder_framework/form"
)

// ContactForm is the form struct with validation.
type ContactForm struct {
	Name    string `form:"name" validate:"required"`
	Email   string `form:"email" validate:"required"`
	Message string `form:"message" validate:"required"`
}

var Comp = component.Component{
	TemplatePath: componentDir() + "/contact.html",
	LayoutPath:   layoutDir() + "/layout.html",
	StylePath:    componentDir() + "/contact.css",
	Handler: func(ctx *component.Ctx) any {
		result := ctx.SessionState.Get("form_result")
		if result != nil {
			ctx.SessionState.Set("form_result", nil)
			return result
		}
		return map[string]any{
			"Errors":  map[string]string{},
			"Success": false,
			"Form":    ContactForm{},
		}
	},
}

// Register registers the contact form component and its submit action.
func Register(app *thunder.App) {
	app.Component("/", Comp)

	app.Action("/submit", Comp, func(ctx *component.Ctx) {
		data, err := component.FormDecode[ContactForm](ctx)
		if err != nil {
			if ve, ok := err.(form.ValidationError); ok {
				errors := make(map[string]string)
				for _, fe := range ve.Errors {
					errors[fe.Field] = fe.Message
				}
				ctx.SessionState.Set("form_result", map[string]any{
					"Errors":  errors,
					"Success": false,
					"Form":    data,
				})
				return
			}
			app.Logger.Error("form decode error: " + err.Error())
			return
		}

		app.Logger.Info("Contact form submitted: " + data.Name + " <" + data.Email + ">")
		ctx.SessionState.Set("form_result", map[string]any{
			"Errors":  map[string]string{},
			"Success": true,
			"Form":    ContactForm{},
		})
	})
}

func componentDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/contact-form/components"
}

func layoutDir() string {
	dir, _ := os.Getwd()
	return dir + "/examples/contact-form/components/layout"
}
