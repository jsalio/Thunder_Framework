package component

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// New creates a Component by auto-detecting co-located files.
//
// It uses runtime.Caller to find the directory of the calling .go file,
// then resolves template and style paths by convention:
//   - Template: same directory, same base name as the .go file, with .html extension
//   - Style:    same directory, same base name as the .go file, with .css extension (optional)
//
// Example:
//
//	// In components/counter/counter.go:
//	var Comp = component.New(handler)
//	// Auto-resolves: counter.html, counter.css (if exists)
func New(handler func(ctx *Ctx) any) Component {
	_, callerFile, _, _ := runtime.Caller(1)
	dir := filepath.Dir(callerFile)
	base := strings.TrimSuffix(filepath.Base(callerFile), ".go")

	templatePath := filepath.Join(dir, base+".html")
	stylePath := filepath.Join(dir, base+".css")

	// Only set style path if the file actually exists.
	if _, err := os.Stat(stylePath); err != nil {
		stylePath = ""
	}

	return Component{
		TemplatePath: templatePath,
		StylePath:    stylePath,
		Handler:      handler,
		dir:          dir,
	}
}

// WithLayout sets the layout path, resolved relative to the component's directory.
//
//	var Comp = component.New(handler).WithLayout("../layout/layout.html")
func (c Component) WithLayout(layoutPath string) Component {
	if !filepath.IsAbs(layoutPath) {
		layoutPath = filepath.Join(c.dir, layoutPath)
	}
	c.LayoutPath = layoutPath
	return c
}

// WithStyle overrides the auto-detected style path.
func (c Component) WithStyle(stylePath string) Component {
	if !filepath.IsAbs(stylePath) {
		stylePath = filepath.Join(c.dir, stylePath)
	}
	c.StylePath = stylePath
	return c
}

// WithTemplate overrides the auto-detected template path.
func (c Component) WithTemplate(templatePath string) Component {
	if !filepath.IsAbs(templatePath) {
		templatePath = filepath.Join(c.dir, templatePath)
	}
	c.TemplatePath = templatePath
	return c
}

// WithChild registers a named child component for template composition.
// In templates, use {{child "name"}} to render the child inline.
//
//	var Page = component.New(handler).
//	    WithChild("sidebar", sidebar.Comp).
//	    WithChild("footer", footer.Comp)
func (c Component) WithChild(name string, child Component) Component {
	if c.Children == nil {
		c.Children = make(map[string]Component)
	}
	c.Children[name] = child
	return c
}
