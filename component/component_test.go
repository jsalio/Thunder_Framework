package component

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/jsalio/thunder_framework/state"
)

func TestCtx(t *testing.T) {
	s := state.New()
	req := httptest.NewRequest("GET", "/", nil)
	rr := httptest.NewRecorder()
	params := map[string]string{"id": "123"}

	ctx := &Ctx{
		State:   s,
		Request: req,
		Params:  params,
		Writer:  rr,
	}

	if ctx.State != s {
		t.Errorf("expected state mismatch")
	}
	if ctx.Request != req {
		t.Errorf("expected request mismatch")
	}
	if ctx.Params["id"] != "123" {
		t.Errorf("expected param mismatch")
	}
}

func TestComponent(t *testing.T) {
	comp := &Component{
		TemplatePath: "test.html",
		LayoutPath:   "layout.html",
		Handler: func(ctx *Ctx) any {
			return "data"
		},
	}

	if comp.TemplatePath != "test.html" {
		t.Errorf("expected template path mismatch")
	}
	if comp.Handler == nil {
		t.Errorf("expected handler mismatch")
	}
}

func TestNewAutoDetectsPaths(t *testing.T) {
	// New() uses runtime.Caller(1), which points to THIS file: component_test.go
	// So it will look for component_test.html and component_test.css
	handler := func(ctx *Ctx) any { return nil }

	// Create a temporary .html file so the template path resolves
	dir := filepath.Dir(callerFile())
	htmlPath := filepath.Join(dir, "component_test.html")
	os.WriteFile(htmlPath, []byte("<div>test</div>"), 0644)
	defer os.Remove(htmlPath)

	comp := New(handler)

	// Template path should end with component_test.html
	if !strings.HasSuffix(comp.TemplatePath, "component_test.html") {
		t.Errorf("TemplatePath = %q, want suffix component_test.html", comp.TemplatePath)
	}

	// Style should be empty since component_test.css doesn't exist
	if comp.StylePath != "" {
		t.Errorf("StylePath = %q, want empty (no css file)", comp.StylePath)
	}

	// Layout should be empty by default
	if comp.LayoutPath != "" {
		t.Errorf("LayoutPath = %q, want empty", comp.LayoutPath)
	}

	// dir should be set for relative path resolution
	if comp.dir == "" {
		t.Error("dir should be set")
	}
}

func TestNewWithStyleDetection(t *testing.T) {
	handler := func(ctx *Ctx) any { return nil }
	dir := filepath.Dir(callerFile())

	htmlPath := filepath.Join(dir, "component_test.html")
	cssPath := filepath.Join(dir, "component_test.css")
	os.WriteFile(htmlPath, []byte("<div>test</div>"), 0644)
	os.WriteFile(cssPath, []byte(".test{}"), 0644)
	defer os.Remove(htmlPath)
	defer os.Remove(cssPath)

	comp := New(handler)

	if !strings.HasSuffix(comp.StylePath, "component_test.css") {
		t.Errorf("StylePath = %q, want suffix component_test.css", comp.StylePath)
	}
}

func TestWithLayout(t *testing.T) {
	handler := func(ctx *Ctx) any { return nil }
	dir := filepath.Dir(callerFile())

	htmlPath := filepath.Join(dir, "component_test.html")
	os.WriteFile(htmlPath, []byte("<div>test</div>"), 0644)
	defer os.Remove(htmlPath)

	comp := New(handler).WithLayout("../layout/layout.html")

	// Should resolve relative to the component's directory
	if !strings.Contains(comp.LayoutPath, "layout") {
		t.Errorf("LayoutPath = %q, want to contain 'layout'", comp.LayoutPath)
	}
	if !filepath.IsAbs(comp.LayoutPath) {
		t.Errorf("LayoutPath = %q, should be absolute", comp.LayoutPath)
	}
}

func TestWithChild(t *testing.T) {
	handler := func(ctx *Ctx) any { return nil }
	dir := filepath.Dir(callerFile())

	htmlPath := filepath.Join(dir, "component_test.html")
	os.WriteFile(htmlPath, []byte("<div>parent</div>"), 0644)
	defer os.Remove(htmlPath)

	childComp := Component{TemplatePath: "child.html", Handler: handler}
	comp := New(handler).WithChild("sidebar", childComp)

	if comp.Children == nil {
		t.Fatal("Children map should not be nil")
	}
	if _, ok := comp.Children["sidebar"]; !ok {
		t.Error("expected child 'sidebar' to be registered")
	}
}

func TestWithChildMultiple(t *testing.T) {
	handler := func(ctx *Ctx) any { return nil }
	dir := filepath.Dir(callerFile())

	htmlPath := filepath.Join(dir, "component_test.html")
	os.WriteFile(htmlPath, []byte("<div>parent</div>"), 0644)
	defer os.Remove(htmlPath)

	comp := New(handler).
		WithChild("header", Component{TemplatePath: "header.html"}).
		WithChild("footer", Component{TemplatePath: "footer.html"})

	if len(comp.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(comp.Children))
	}
}

// callerFile returns the path of this test file via runtime.Caller.
func callerFile() string {
	_, file, _, _ := runtime.Caller(0)
	return file
}
