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
