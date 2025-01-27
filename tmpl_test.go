package tmpl

import (
	"bytes"
	"testing"
	"testing/fstest"
)

func TestLoad(t *testing.T) {
	fs := fstest.MapFS{
		"test.html": {
			Data: []byte("<p>Test html</p>"),
		},
		"test.tmpl": {
			Data: []byte("<p>Test tmpl</p>"),
		},
		"test": {
			Data: []byte("<p>Test</p>"),
		},
	}
	buf := new(bytes.Buffer)
	tests := []struct {
		templates Templates
		template  Template
		expected  string
	}{
		{
			templates: New(fs).Load("test").MustParse(),
			template:  Tmpl("test"),
			expected:  "<p>Test html</p>",
		},
		{
			templates: New(fs).SetExt("html").Load("test").MustParse(),
			template:  Tmpl("test"),
			expected:  "<p>Test html</p>",
		},
		{
			templates: New(fs).SetExt("tmpl").Load("test").MustParse(),
			template:  Tmpl("test"),
			expected:  "<p>Test tmpl</p>",
		},
		{
			templates: New(fs).SetExt("").Load("test").MustParse(),
			template:  Tmpl("test"),
			expected:  "<p>Test</p>",
		},
	}
	for _, test := range tests {
		err := test.templates.Render(buf, test.template)
		if err != nil {
			t.Error(err)
		}
		if buf.String() != test.expected {
			t.Errorf("expected: %q, got: %q", test.expected, buf.String())
		}
		buf.Reset()
	}
}

func TestLoadTree(t *testing.T) {
	fs := fstest.MapFS{
		"layout.html": {
			Data: []byte(`<h1>{{ .Data }}</h1>{{ template "content" .Child }}`),
		},
		"index.html": {
			Data: []byte(`{{ template "layout" . }}
			{{- define "content" }}<p>{{ . }}</p>{{ end -}}
			`),
		},
		"sub/layout.html": {
			Data: []byte(`{{ template "layout" . }}
			{{- define "content" }}<h2>{{ .Data }}</h2>{{ template "sub/content" .Child }}{{ end -}}
			`),
		},
		"sub/index.html": {
			Data: []byte(`{{ template "sub/layout" . }}
			{{- define "sub/content" }}<p>{{ . }}</p>{{ end -}}
			`),
		},
	}
	buf := new(bytes.Buffer)
	templates := New(fs).LoadTree(".").MustParse()
	tests := []struct {
		template Template
		expected string
	}{
		{
			template: Tmpl("index", 1, 2),
			expected: "<h1>1</h1><p>2</p>",
		},
		{
			template: Tmpl("sub/index", 1, 2, 3),
			expected: "<h1>1</h1><h2>2</h2><p>3</p>",
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

func TestLoadTreeNested(t *testing.T) {
	fs := fstest.MapFS{
		"layout.html": {
			Data: []byte(`<h1>{{ .Data }}</h1>{{ template "content" .Child }}`),
		},
		"index.html": {
			Data: []byte(`{{ template "layout" . }}
			{{- define "content" }}<p>{{ . }}</p>{{ end -}}
			`),
		},
		"sub/layout.html": {
			Data: []byte(`{{ template "layout" . }}
			{{- define "content" }}<h2>{{ .Data }}</h2>{{ template "sub/content" .Child }}{{ end -}}
			`),
		},
		"sub/index.html": {
			Data: []byte(`{{ template "sub/layout" . }}
			{{- define "sub/content" }}<p>{{ . }}</p>{{ end -}}
			`),
		},
		"nosub/layout.html": {
			Data: []byte(`<h2>{{ .Data }}</h2>{{ template "nosub/content" .Child }}`),
		},
		"nosub/index.html": {
			Data: []byte(`{{ template "nosub/layout" . }}
			{{- define "nosub/content" }}<p>{{ . }}</p>{{ end -}}
			`),
		},
	}
	buf := new(bytes.Buffer)
	templates := New(fs).LoadTree("sub").LoadTree("nosub").MustParse()
	tests := []struct {
		template    Template
		expected    string
		shouldError bool
	}{
		{
			template:    Tmpl("index", 1, 2),
			expected:    "",
			shouldError: true,
		},
		{
			template: Tmpl("sub/index", 1, 2, 3),
			expected: "<h1>1</h1><h2>2</h2><p>3</p>",
		},
		{
			template: Tmpl("nosub/index", 1, 2),
			expected: "<h2>1</h2><p>2</p>",
		},
	}
	for _, test := range tests {
		err := templates.Render(buf, test.template)
		if test.shouldError {
			if err == nil {
				name, _ := test.template.Template()
				t.Error("expected error while rendering", name)
			}
		} else {
			if err != nil {
				t.Error(err)
			}
		}
		if buf.String() != test.expected {
			t.Errorf("expected: %q, got: %q", test.expected, buf.String())
		}
		buf.Reset()
	}
}

func TestLoadTreeWithoutLayout(t *testing.T) {
	fs := fstest.MapFS{
		"index.html": {
			Data: []byte(`<p>{{ . }}</p>`),
		},
		"sub/index.html": {
			Data: []byte(`<p>{{ . }}</p>`),
		},
	}
	buf := new(bytes.Buffer)
	templates := New(fs).LoadTree(".").MustParse()
	tests := []struct {
		template Template
		expected string
	}{
		{
			template: Tmpl("index", 1),
			expected: "<p>1</p>",
		},
		{
			template: Tmpl("sub/index", 2),
			expected: "<p>2</p>",
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
