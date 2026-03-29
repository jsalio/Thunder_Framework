package component

import (
	"net/http"
	"thunder/internal/state"
)

// Ctx es el contexto que se pasa al Handler de un componente.
// Equivale al injection context de Angular: accede al estado global,
// a la request y a los parámetros de ruta.
type Ctx struct {
	State   *state.State
	Request *http.Request
	Params  map[string]string
	Writer  http.ResponseWriter
}

// Component une el template HTML con su handler de datos.
// Equivale a un @Component de Angular: lógica + vista co-locados.
type Component struct {
	// TemplatePath es la ruta al archivo .html del componente.
	// Debe ser relativa al directorio de trabajo o absoluta.
	TemplatePath string

	// LayoutPath es la ruta opcional al layout envolvente.
	// Si está vacío, el componente se renderiza sin layout (partial).
	LayoutPath string

	// Handler es la función que provee los datos al template.
	// Retorna cualquier valor que se pasará como data al template.
	Handler func(ctx *Ctx) any
}
