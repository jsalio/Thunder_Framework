package render

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderFile(t *testing.T) {
	tmpDir := t.TempDir()

	layoutPath := filepath.Join(tmpDir, "layout.html")
	pagePath := filepath.Join(tmpDir, "page.html")

	layoutContent := `<html><body>{{template "content" .}}</body></html>`
	pageContent := `{{define "content"}}Hello {{.}}!{{end}}`

	if err := os.WriteFile(layoutPath, []byte(layoutContent), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
		t.Fatal(err)
	}

	engine := New(tmpDir, ".html", true)

	t.Run("WithLayout", func(t *testing.T) {
		var buf bytes.Buffer
		err := engine.RenderFile(&buf, pagePath, layoutPath, "", "World")
		if err != nil {
			t.Fatalf("RenderFile failed: %v", err)
		}
		expected := `<html><body>Hello World!</body></html>`
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})

	t.Run("NoLayout", func(t *testing.T) {
		var buf bytes.Buffer
		err := engine.RenderFile(&buf, pagePath, "", "", "World")
		if err != nil {
			t.Fatalf("RenderFile failed: %v", err)
		}
		expected := `Hello World!`
		if buf.String() != expected {
			t.Errorf("expected %q, got %q", expected, buf.String())
		}
	})
}

func TestRenderCaching(t *testing.T) {
	tmpDir := t.TempDir()
	pagePath := filepath.Join(tmpDir, "page.html")
	pageContent := `{{define "content"}}V1{{end}}`
	if err := os.WriteFile(pagePath, []byte(pageContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Caching enabled (isDebug = false)
	engine := New(tmpDir, ".html", false)

	var buf bytes.Buffer
	engine.RenderFile(&buf, pagePath, "", "", nil)
	if buf.String() != "V1" {
		t.Errorf("expected V1, got %q", buf.String())
	}

	// Modify file
	if err := os.WriteFile(pagePath, []byte(`{{define "content"}}V2{{end}}`), 0644); err != nil {
		t.Fatal(err)
	}

	buf.Reset()
	engine.RenderFile(&buf, pagePath, "", "", nil)
	if buf.String() != "V1" {
		t.Errorf("expected V1 (cached), got %q", buf.String())
	}

	// Debug mode (isDebug = true)
	engineDebug := New(tmpDir, ".html", true)
	buf.Reset()
	engineDebug.RenderFile(&buf, pagePath, "", "", nil)
	if buf.String() != "V2" {
		t.Errorf("expected V2 (reloaded), got %q", buf.String())
	}
}

func TestDefaultFuncMap(t *testing.T) {
	engine := New("", "", true)
	var buf bytes.Buffer
	tmpl, _ := template.New("test").Funcs(engine.funcMap).Parse(`{{year}}`)
	tmpl.Execute(&buf, nil)

	if len(buf.String()) != 4 {
		t.Errorf("expected 4 digit year, got %q", buf.String())
	}
}
