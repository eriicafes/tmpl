package tmpl

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestSlot(t *testing.T) {
	fs := fstest.MapFS{
		"counter.html": {
			Data: []byte(`
			{{- template "partials/button" map
				"disabled" (gt . 100)
				"invalid" (lt . 0)
				"children" (tmpl "content" .)
				"errorChildren" "<span>Invalid count</span>"
			-}}
			{{- define "content" }}<span>Count: {{ . }}</span>{{ end -}}
			`),
		},
		"partials/button.html": {
			Data: []byte(`<button{{ if .disabled }} disabled{{ end }}>
			{{- if .invalid }}
			{{- slot .errorChildren -}}
			{{ else }}
			{{- slot .children -}}
			{{ end -}}
			</button>`),
		},
	}
	buf := new(bytes.Buffer)
	templates := New(fs).Load("partials/button", "counter").MustParse()
	tests := []struct {
		template Template
		expected string
	}{
		{
			template: Tmpl("counter", 1),
			expected: "<button><span>Count: 1</span></button>",
		},
		{
			template: Tmpl("counter", 101),
			expected: "<button disabled><span>Count: 101</span></button>",
		},
		{
			template: Tmpl("counter", -1),
			expected: "<button>&lt;span&gt;Invalid count&lt;/span&gt;</button>",
		},
	}
	for _, test := range tests {
		err := templates.Render(buf, test.template)
		if err != nil {
			t.Error(err)
		}
		if buf.String() != test.expected {
			t.Errorf("expected: %q, got: %q", test.expected, buf.String())
		}
		buf.Reset()
	}
}
