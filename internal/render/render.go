package render

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Engine struct {
	directory string
	extension string
	cache     map[string]*template.Template
	mu        sync.RWMutex
	funcMap   template.FuncMap
	isDebug   bool
}

func (e *Engine) SetDirectory(dir string) {
	e.directory = dir
}

func New(directory string, extension string, isDebug bool) *Engine {
	return &Engine{
		directory: directory,
		extension: extension,
		cache:     make(map[string]*template.Template),
		funcMap:   defaultFuncMap(),
		isDebug:   isDebug,
	}
}

func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcMap[name] = fn
}

func (e *Engine) SetDebug(isDebug bool) {
	e.isDebug = isDebug
}

// Render renderiza una página por nombre (modo legacy con directorio base).
func (e *Engine) Render(buffer io.Writer, name string, data any) error {
	template, err := e.getTemplate(name)
	if err != nil {
		return fmt.Errorf("render %q: %w", name, err)
	}
	return template.Execute(buffer, data)
}

// RenderFile renderiza un componente desde su path absoluto/relativo,
// opcionalmente envuelto en un layout. Este es el método principal del
// sistema de componentes co-locados.
func (e *Engine) RenderFile(buffer io.Writer, templatePath, layoutPath string, data any) error {
	cacheKey := templatePath + "|" + layoutPath

	// 1. Buscar en cache
	var tmpl *template.Template
	if !e.isDebug {
		e.mu.RLock()
		cached, ok := e.cache[cacheKey]
		e.mu.RUnlock()
		if ok {
			tmpl = cached
		}
	}

	// 2. Compilar si no estaba en cache
	if tmpl == nil {
		if _, err := os.Stat(templatePath); os.IsNotExist(err) {
			return fmt.Errorf("template %q no encontrado", templatePath)
		}

		var err error
		if layoutPath != "" {
			if _, err := os.Stat(layoutPath); os.IsNotExist(err) {
				return fmt.Errorf("layout %q no encontrado", layoutPath)
			}
			// El layout es el template raíz; la página se define dentro.
			tmpl, err = template.New(filepath.Base(layoutPath)).
				Funcs(e.funcMap).
				ParseFiles(layoutPath, templatePath)
		} else {
			// Sin layout: renderiza solo el fragmento del componente.
			tmpl, err = template.New(filepath.Base(templatePath)).
				Funcs(e.funcMap).
				ParseFiles(templatePath)
		}

		if err != nil {
			return fmt.Errorf("compilando template: %w", err)
		}

		if !e.isDebug {
			e.mu.Lock()
			e.cache[cacheKey] = tmpl
			e.mu.Unlock()
		}
	}

	// 3. Ejecutar — un solo camino para cache hit y miss.
	// Sin layout: ejecutar el bloque "content" directamente,
	// porque el template raíz está vacío (todo vive dentro de {{define "content"}}).
	if layoutPath == "" {
		return tmpl.ExecuteTemplate(buffer, "content", data)
	}
	return tmpl.Execute(buffer, data)
}

// RenderPartial renderiza solo el fragmento de un componente sin layout.
// Útil para respuestas parciales (HTMX, fetch, etc.).
func (e *Engine) RenderPartial(buffer io.Writer, templatePath string, data any) error {
	return e.RenderFile(buffer, templatePath, "", data)
}

// getTemplate mantiene compatibilidad con el sistema legacy de directorios.
func (e *Engine) getTemplate(name string) (*template.Template, error) {
	if !e.isDebug {
		e.mu.RLock()
		if t, ok := e.cache[name]; ok {
			e.mu.RUnlock()
			return t, nil
		}
		e.mu.RUnlock()
	}
	layoutPath := filepath.Join(e.directory, "layout", "*"+e.extension)
	pagePath := filepath.Join(e.directory, "pages", name+e.extension)

	if _, err := os.Stat(pagePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("plantilla %q no encontrada en %s", name, pagePath)
	}

	var tmpl *template.Template
	var err error

	layoutFiles, globErr := filepath.Glob(layoutPath)
	if globErr == nil && len(layoutFiles) > 0 {
		files := append(layoutFiles, pagePath)
		tmpl, err = template.New("layout"+e.extension).Funcs(e.funcMap).ParseFiles(files...)
	} else {
		tmpl, err = template.New(name + e.extension).Funcs(e.funcMap).ParseFiles(pagePath)
	}

	if err != nil {
		return nil, fmt.Errorf("compilando plantilla: %w", err)
	}

	if !e.isDebug {
		e.mu.Lock()
		e.cache[name] = tmpl
		e.mu.Unlock()
	}

	return tmpl, nil
}

func defaultFuncMap() template.FuncMap {
	return template.FuncMap{
		"safeHTML": safeHTML,
		"safeJS":   secureJS,
		"safeCSS":  secureCSS,
		"safeURL":  secureURL,
		"safeAttr": secureAttr,
		"year":     year,
	}
}

func year() int {
	return time.Now().Year()
}

func safeHTML(s string) template.HTML {
	return template.HTML(s)
}

func secureJS(s string) template.JS {
	return template.JS(s)
}

func secureCSS(s string) template.CSS {
	return template.CSS(s)
}

func secureURL(s string) template.URL {
	return template.URL(s)
}

func secureAttr(s string) template.HTMLAttr {
	return template.HTMLAttr(s)
}
