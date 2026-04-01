package render

import (
	"testing"
)

func TestPreprocessFor(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic for",
			in:   `<li t-for=".Items">{{.Name}}</li>`,
			want: `{{range .Items}}<li>{{.Name}}</li>{{end}}`,
		},
		{
			name: "for on template strips tags",
			in:   `<template t-for=".Items"><dt>{{.K}}</dt><dd>{{.V}}</dd></template>`,
			want: `{{range .Items}}<dt>{{.K}}</dt><dd>{{.V}}</dd>{{end}}`,
		},
		{
			name: "nested same tags",
			in:   `<div t-for=".Items"><div>inner</div></div>`,
			want: `{{range .Items}}<div><div>inner</div></div>{{end}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Preprocess(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestPreprocessIf(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic if",
			in:   `<div t-if=".Active">yes</div>`,
			want: `{{if .Active}}<div>yes</div>{{end}}`,
		},
		{
			name: "if with not",
			in:   `<p t-if="not .Items">empty</p>`,
			want: `{{if not .Items}}<p>empty</p>{{end}}`,
		},
		{
			name: "if on template",
			in:   `<template t-if=".X"><a>1</a><b>2</b></template>`,
			want: `{{if .X}}<a>1</a><b>2</b>{{end}}`,
		},
		{
			name: "if with single quotes",
			in:   `<form t-if='gt (index .Stats "Done") 0'>btn</form>`,
			want: `{{if gt (index .Stats "Done") 0}}<form>btn</form>{{end}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Preprocess(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestPreprocessIfElse(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "if-else same tag",
			in:   `<p t-if=".X">yes</p>` + "\n" + `<p t-else>no</p>`,
			want: `{{if .X}}<p>yes</p>{{else}}` + "\n" + `<p>no</p>{{end}}`,
		},
		{
			name: "if-else with template",
			in:   `<li t-if="not .Items">empty</li>` + "\n" + `<template t-else><li>has</li></template>`,
			want: `{{if not .Items}}<li>empty</li>{{else}}` + "\n" + `<li>has</li>{{end}}`,
		},
		{
			name: "if-else-if",
			in:   `<p t-if=".A">a</p>` + "\n" + `<p t-else-if=".B">b</p>`,
			want: `{{if .A}}<p>a</p>{{else if .B}}` + "\n" + `<p>b</p>{{end}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Preprocess(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestPreprocessClassDirective(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "add conditional class",
			in:   `<li class="item" t-class-active=".Done">text</li>`,
			want: `<li class="item{{if .Done}} active{{end}}">text</li>`,
		},
		{
			name: "multiple conditional classes",
			in:   `<div class="base" t-class-a=".X" t-class-b=".Y">ok</div>`,
			want: `<div class="base{{if .X}} a{{end}}{{if .Y}} b{{end}}">ok</div>`,
		},
		{
			name: "no existing class attr",
			in:   `<span t-class-highlight=".Active">text</span>`,
			want: `<span class="{{if .Active}} highlight{{end}}">text</span>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Preprocess(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestPreprocessMorph(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "basic t-morph adds both attributes",
			in:   `<div class="widget" t-morph>content</div>`,
			want: `<div class="widget" hx-swap="morph:innerHTML" hx-ext="morph">content</div>`,
		},
		{
			name: "t-morph with existing hx-swap prefixes value",
			in:   `<form t-morph hx-post="/add" hx-swap="innerHTML">ok</form>`,
			want: `<form hx-post="/add" hx-swap="morph:innerHTML" hx-ext="morph">ok</form>`,
		},
		{
			name: "t-morph with existing hx-ext merges",
			in:   `<div t-morph hx-ext="other">ok</div>`,
			want: `<div hx-ext="other morph" hx-swap="morph:innerHTML">ok</div>`,
		},
		{
			name: "t-morph idempotent on already-prefixed swap",
			in:   `<div t-morph hx-swap="morph:outerHTML">ok</div>`,
			want: `<div hx-swap="morph:outerHTML" hx-ext="morph">ok</div>`,
		},
		{
			name: "self-closing tag",
			in:   `<input t-morph/>`,
			want: `<input hx-swap="morph:innerHTML" hx-ext="morph"/>`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Preprocess(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}

func TestPreprocessMorphCombined(t *testing.T) {
	in := `<li t-for=".Items" t-morph hx-swap="innerHTML">{{.Name}}</li>`
	want := `{{range .Items}}<li hx-swap="morph:innerHTML" hx-ext="morph">{{.Name}}</li>{{end}}`
	got := Preprocess(in)
	if got != want {
		t.Errorf("\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPreprocessCombined(t *testing.T) {
	in := `<li t-for=".Todos" class="item" t-class-done=".Done">{{.Text}}</li>`
	want := `{{range .Todos}}<li class="item{{if .Done}} done{{end}}">{{.Text}}</li>{{end}}`
	got := Preprocess(in)
	if got != want {
		t.Errorf("\ngot:  %s\nwant: %s", got, want)
	}
}

func TestPreprocessPassthrough(t *testing.T) {
	// Sin directivas: el output debe ser idéntico al input
	in := `{{define "content"}}<div class="test">{{.Name}}</div>{{end}}`
	got := Preprocess(in)
	if got != in {
		t.Errorf("expected passthrough, got: %s", got)
	}
}

func TestPreprocessPage(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "auto-wrap with t-title",
			in:   "<t-title>My Page</t-title>\n\n<div>hello</div>",
			want: "{{define \"title\"}}My Page{{end}}\n\n{{define \"content\"}}\n<div>hello</div>\n{{end}}\n",
		},
		{
			name: "auto-wrap without t-title",
			in:   "<div>hello</div>",
			want: "{{define \"content\"}}\n<div>hello</div>\n{{end}}\n",
		},
		{
			name: "backwards compat with existing define",
			in:   "{{define \"title\"}}X{{end}}\n{{define \"content\"}}<p>old</p>{{end}}",
			want: "{{define \"title\"}}X{{end}}\n{{define \"content\"}}<p>old</p>{{end}}",
		},
		{
			name: "t-title with whitespace",
			in:   "<t-title>  Spaced Title  </t-title>\n<p>body</p>",
			want: "{{define \"title\"}}Spaced Title{{end}}\n\n{{define \"content\"}}\n<p>body</p>\n{{end}}\n",
		},
		{
			name: "directives still processed",
			in:   "<t-title>App</t-title>\n<p t-if=\".X\">yes</p>",
			want: "{{define \"title\"}}App{{end}}\n\n{{define \"content\"}}\n{{if .X}}<p>yes</p>{{end}}\n{{end}}\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessPage(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestInjectFrameworkScripts(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "injects before </body>",
			in:   "<html><body><main>content</main></body></html>",
			want: "<html><body><main>content</main>" + frameworkScripts + "</body></html>",
		},
		{
			name: "no body tag passes through",
			in:   "<div>fragment</div>",
			want: "<div>fragment</div>",
		},
		{
			name: "already has __thunder skips injection",
			in:   `<body><script src="/__thunder/htmx.min.js"></script></body>`,
			want: `<body><script src="/__thunder/htmx.min.js"></script></body>`,
		},
		{
			name: "case insensitive body tag",
			in:   "<html><BODY>hi</BODY></html>",
			want: "<html><BODY>hi" + frameworkScripts + "</BODY></html>",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := injectFrameworkScripts(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestPreprocessLayout(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "strip layout define wrapper",
			in:   "{{define \"layout.html\"}}\n<html>{{template \"content\" .}}</html>\n{{end}}\n",
			want: "\n<html>{{template \"content\" .}}</html>\n",
		},
		{
			name: "no wrapper passes through",
			in:   "<html>{{template \"content\" .}}</html>",
			want: "<html>{{template \"content\" .}}</html>",
		},
		{
			name: "inner blocks preserved",
			in:   "{{define \"layout.html\"}}\n<head>{{block \"styles\" .}}{{end}}</head>\n{{end}}",
			want: "\n<head>{{block \"styles\" .}}{{end}}</head>\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PreprocessLayout(tt.in)
			if got != tt.want {
				t.Errorf("\ngot:  %q\nwant: %q", got, tt.want)
			}
		})
	}
}
