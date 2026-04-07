package render

import (
	"testing"
)

func TestPreprocessComponents(t *testing.T) {
	known := map[string]bool{
		"counter":   true,
		"sidebar":   true,
		"user-card": true,
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "self-closing tag",
			in:   `<div><t-counter /></div>`,
			want: `<div>{{component "counter"}}</div>`,
		},
		{
			name: "tag with closing pair",
			in:   `<div><t-sidebar>ignored content</t-sidebar></div>`,
			want: `<div>{{component "sidebar"}}</div>`,
		},
		{
			name: "hyphenated component name",
			in:   `<t-user-card />`,
			want: `{{component "user-card"}}`,
		},
		{
			name: "multiple components",
			in:   `<t-counter /><t-sidebar />`,
			want: `{{component "counter"}}{{component "sidebar"}}`,
		},
		{
			name: "unknown component left untouched",
			in:   `<t-unknown />`,
			want: `<t-unknown />`,
		},
		{
			name: "t-if not treated as component",
			in:   `<div t-if=".Active">yes</div>`,
			want: `<div t-if=".Active">yes</div>`,
		},
		{
			name: "t-title not treated as component",
			in:   `<t-title>My Page</t-title>`,
			want: `<t-title>My Page</t-title>`,
		},
		{
			name: "empty known set leaves everything",
			in:   `<t-counter />`,
			want: `<t-counter />`,
		},
		{
			name: "component mixed with regular HTML",
			in:   `<h1>Hello</h1><t-counter /><p>World</p>`,
			want: `<h1>Hello</h1>{{component "counter"}}<p>World</p>`,
		},
		{
			name: "self-closing without space",
			in:   `<t-counter/>`,
			want: `{{component "counter"}}`,
		},
		{
			name: "component with t-sse",
			in:   `<t-counter t-sse="count-changed" />`,
			want: `<div data-t-sse="count-changed" data-t-sse-url="/__thunder/component/counter">{{component "counter"}}</div>`,
		},
		{
			name: "component with t-sse and other content",
			in:   `<div><t-counter t-sse="upd" /></div>`,
			want: `<div><div data-t-sse="upd" data-t-sse-url="/__thunder/component/counter">{{component "counter"}}</div></div>`,
		},
		{
			name: "regular tag with t-sse",
			in:   `<div t-sse="refresh-me">content</div>`,
			want: `<div data-t-sse="refresh-me">content</div>`,
		},
		{
			name: "regular tag with t-sse and trailing slash",
			in:   `<img src="foo.png" t-sse="reload" />`,
			want: `<img src="foo.png" data-t-sse="reload"/>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := known
			if tt.name == "empty known set leaves everything" {
				k = nil
			}
			got := PreprocessComponents(tt.in, k)
			if got != tt.want {
				t.Errorf("\ngot:  %s\nwant: %s", got, tt.want)
			}
		})
	}
}
