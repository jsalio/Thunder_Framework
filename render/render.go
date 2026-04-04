// Package render provides the template engine for the framework.
// It supports layout wrapping, partial rendering, and caching.
package render

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Engine is the template rendering engine.
type Engine struct {
	directory string
	extension string
	cache     map[string]*template.Template
	mu        sync.RWMutex
	cssCache  map[string]string
	cssMu     sync.RWMutex
	funcMap   template.FuncMap
	isDebug   bool
}

// SetDirectory sets the base directory for templates.
func (e *Engine) SetDirectory(dir string) {
	e.directory = dir
}

// New creates and initializes a new template Engine.
func New(directory string, extension string, isDebug bool) *Engine {
	return &Engine{
		directory: directory,
		extension: extension,
		cache:     make(map[string]*template.Template),
		cssCache:  make(map[string]string),
		funcMap:   defaultFuncMap(),
		isDebug:   isDebug,
	}
}

// AddFunc registers a global function for use in templates.
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.funcMap[name] = fn
}

// SetDebug toggles debug mode. In debug mode, templates are recompiled on every request.
func (e *Engine) SetDebug(isDebug bool) {
	e.isDebug = isDebug
}

// Render renders a page by its name (legacy mode with base directory).
func (e *Engine) Render(buffer io.Writer, name string, data any) error {
	template, err := e.getTemplate(name)
	if err != nil {
		return fmt.Errorf("render %q: %w", name, err)
	}
	return template.Execute(buffer, data)
}

// RenderFile renders a component from its absolute/relative path,
// optionally wrapped in a layout. If stylePath is set, the CSS is
// injected as an inline <style> tag (in <head> for layouts, before
// content for partials).
// RenderFileWithCSRF is like RenderFile but binds a CSRF token for template use.
func (e *Engine) RenderFileWithCSRF(buffer io.Writer, templatePath, layoutPath, stylePath string, data any, csrfToken string) error {
	return e.renderFile(buffer, templatePath, layoutPath, stylePath, data, csrfToken, nil)
}

func (e *Engine) RenderFile(buffer io.Writer, templatePath, layoutPath, stylePath string, data any) error {
	return e.renderFile(buffer, templatePath, layoutPath, stylePath, data, "", nil)
}

// RenderWithFuncs renders a component with additional per-request template functions.
// Used by the child composition system to inject {{child "name"}} into templates.
func (e *Engine) RenderWithFuncs(buffer io.Writer, templatePath, layoutPath, stylePath string, data any, csrfToken string, extraFuncs template.FuncMap) error {
	return e.renderFile(buffer, templatePath, layoutPath, stylePath, data, csrfToken, extraFuncs)
}

func (e *Engine) renderFile(buffer io.Writer, templatePath, layoutPath, stylePath string, data any, csrfToken string, extraFuncs template.FuncMap) error {
	cacheKey := templatePath + "|" + layoutPath + "|" + stylePath

	// 1. Search in cache
	var tmpl *template.Template
	if !e.isDebug {
		e.mu.RLock()
		cached, ok := e.cache[cacheKey]
		e.mu.RUnlock()
		if ok {
			tmpl = cached
		}
	}

	// 2. Compile if not in cache
	if tmpl == nil {
		pageSrc, err := readAndPreprocessPage(templatePath)
		if err != nil {
			return err
		}

		if layoutPath != "" {
			layoutSrc, err := readAndPreprocessLayout(layoutPath)
			if err != nil {
				return err
			}
			// The layout is the root template; the page is defined inside.
			tmpl, err = template.New(filepath.Base(layoutPath)).
				Funcs(e.funcMap).
				Parse(layoutSrc)
			if err != nil {
				return fmt.Errorf("compiling layout: %w", err)
			}
			if _, err = tmpl.New(filepath.Base(templatePath)).Parse(pageSrc); err != nil {
				return fmt.Errorf("compiling template: %w", err)
			}
		} else {
			tmpl, err = template.New(filepath.Base(templatePath)).
				Funcs(e.funcMap).
				Parse(pageSrc)
			if err != nil {
				return fmt.Errorf("compiling template: %w", err)
			}
		}

		// Inject component CSS as a "component-styles" block for layouts.
		// The layout uses {{block "component-styles" .}}{{end}} in <head>.
		if layoutPath != "" && stylePath != "" {
			css, cssErr := e.loadCSS(stylePath)
			if cssErr != nil {
				return cssErr
			}
			if css != "" {
				tmpl.Funcs(template.FuncMap{
					"__css": func() template.CSS { return template.CSS(css) },
				})
				if _, err = tmpl.Parse(`{{define "component-styles"}}<style>{{__css}}</style>{{end}}`); err != nil {
					return fmt.Errorf("injecting component CSS: %w", err)
				}
			}
		}

		if !e.isDebug {
			e.mu.Lock()
			e.cache[cacheKey] = tmpl
			e.mu.Unlock()
		}
	}

	// 3. Execute — same path for cache hit and miss.
	// Clone the template for per-request functions (CSRF token, child components).
	execTmpl := tmpl
	if csrfToken != "" || len(extraFuncs) > 0 {
		cloned, err := tmpl.Clone()
		if err != nil {
			return fmt.Errorf("cloning template: %w", err)
		}
		funcs := template.FuncMap{}
		if csrfToken != "" {
			funcs["csrfToken"] = func() string { return csrfToken }
		}
		for k, v := range extraFuncs {
			funcs[k] = v
		}
		cloned.Funcs(funcs)
		execTmpl = cloned
	}

	if layoutPath == "" {
		// Partial render: inject CSS inline before content.
		if stylePath != "" {
			css, err := e.loadCSS(stylePath)
			if err != nil {
				return err
			}
			if css != "" {
				io.WriteString(buffer, "<style>")
				io.WriteString(buffer, css)
				io.WriteString(buffer, "</style>\n")
			}
		}
		return execTmpl.ExecuteTemplate(buffer, "content", data)
	}
	// Full render: CSS is already in the compiled template via "component-styles" block.
	return execTmpl.Execute(buffer, data)
}

// RenderPartialToString renders a component as a partial (no layout) and returns the HTML string.
// Used internally for child component composition — the parent template calls {{child "name"}}
// which triggers this to render the child inline.
func (e *Engine) RenderPartialToString(templatePath, stylePath string, data any, csrfToken string) (string, error) {
	var buf bytes.Buffer
	if err := e.renderFile(&buf, templatePath, "", stylePath, data, csrfToken, nil); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderPartial renders only a component fragment without a layout.
// Useful for partial responses (HTMX, fetch, etc.).
func (e *Engine) RenderPartial(buffer io.Writer, templatePath, stylePath string, data any) error {
	return e.renderFile(buffer, templatePath, "", stylePath, data, "", nil)
}

// RenderPartialWithCSRF is like RenderPartial but binds a CSRF token.
func (e *Engine) RenderPartialWithCSRF(buffer io.Writer, templatePath, stylePath string, data any, csrfToken string) error {
	return e.renderFile(buffer, templatePath, "", stylePath, data, csrfToken, nil)
}

// readAndPreprocessPage reads a page/component template and applies the preprocessor.
func readAndPreprocessPage(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template %q: %w", path, err)
	}
	return PreprocessPage(string(content)), nil
}

// readAndPreprocessLayout reads a layout template and applies the preprocessor.
func readAndPreprocessLayout(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading template %q: %w", path, err)
	}
	return PreprocessLayout(string(content)), nil
}

// loadCSS reads and caches a CSS file's content.
func (e *Engine) loadCSS(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	if !e.isDebug {
		e.cssMu.RLock()
		cached, ok := e.cssCache[path]
		e.cssMu.RUnlock()
		if ok {
			return cached, nil
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading CSS %q: %w", path, err)
	}

	css := string(content)

	if !e.isDebug {
		e.cssMu.Lock()
		e.cssCache[path] = css
		e.cssMu.Unlock()
	}

	return css, nil
}

// getTemplate maintains compatibility with the legacy directory system.
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
		return nil, fmt.Errorf("template %q not found at %s", name, pagePath)
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
		return nil, fmt.Errorf("compiling template: %w", err)
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
		"safeHTML":  safeHTML,
		"safeJS":    secureJS,
		"safeCSS":   secureCSS,
		"safeURL":   secureURL,
		"safeAttr":  secureAttr,
		"year":      year,
		"csrfToken": func() string { return "" },
		"child":     func(name string) template.HTML { return "" },
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
