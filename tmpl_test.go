package tmpl

import (
	"bytes"
	"fmt"
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
			template:  Tmpl("test", nil),
			expected:  "<p>Test html</p>",
		},
		{
			templates: New(fs).SetExt("html").Load("test").MustParse(),
			template:  Tmpl("test", nil),
			expected:  "<p>Test html</p>",
		},
		{
			templates: New(fs).SetExt("tmpl").Load("test").MustParse(),
			template:  Tmpl("test", nil),
			expected:  "<p>Test tmpl</p>",
		},
		{
			templates: New(fs).SetExt("").Load("test").MustParse(),
			template:  Tmpl("test", nil),
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

type IndexPage int

func (i IndexPage) Tmpl() Template {
	return Tmpl("index", i)
}

type SubIndexPage int

func (s SubIndexPage) Tmpl() Template {
	return Tmpl("sub/index", s)
}

func TestLoadTree(t *testing.T) {
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
			template: IndexPage(1),
			expected: "<p>1</p>",
		},
		{
			template: Tmpl("sub/index", 2),
			expected: "<p>2</p>",
		},
		{
			template: SubIndexPage(2),
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

type LayoutPage struct {
	Children
	Data int
}

func (l LayoutPage) Tmpl() Template {
	return Associated(l.Base(), "layout", l)
}

type SubLayoutPage struct {
	Children Template
	Data     int
}

func (s SubLayoutPage) Tmpl() Template {
	base, _, _ := Info(s.Children)
	return Associated(base, "sub/layout", s)
}

func TestLoadTreeWithLayout(t *testing.T) {
	fs := fstest.MapFS{
		"layout.html": {
			Data: []byte(`<h1>{{ .Data }}</h1>{{ slot .Children }}`),
		},
		"index.html": {
			Data: []byte(`<p>{{ . }}</p>`),
		},
		"sub/layout.html": {
			Data: []byte(`<h2>{{ .Data }}</h2>{{ slot .Children }}`),
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
			template: Associated(
				"index",
				"layout",
				Map{
					"Data":     1,
					"Children": Tmpl("index", 2),
				},
			),
			expected: "<h1>1</h1><p>2</p>",
		},
		{
			template: LayoutPage{
				Data: 1,
				Children: Children{
					IndexPage(2),
				},
			},
			expected: "<h1>1</h1><p>2</p>",
		},
		{
			template: Wrap(&LayoutPage{Data: 1}, IndexPage(2)),
			expected: "<h1>1</h1><p>2</p>",
		},
		{
			template: Associated(
				"sub/index",
				"layout",
				Map{
					"Data": 1,
					"Children": Tmpl(
						"sub/layout",
						Map{
							"Data":     2,
							"Children": Tmpl("sub/index", 3),
						},
					),
				},
			),
			expected: "<h1>1</h1><h2>2</h2><p>3</p>",
		},
		{
			template: LayoutPage{
				Data: 1,
				Children: Children{
					SubLayoutPage{
						Data:     2,
						Children: SubIndexPage(3),
					},
				},
			},
			expected: "<h1>1</h1><h2>2</h2><p>3</p>",
		},
		{
			template: Wrap(&LayoutPage{Data: 1}, SubLayoutPage{
				Data:     2,
				Children: SubIndexPage(3),
			}),
			expected: "<h1>1</h1><h2>2</h2><p>3</p>",
		},
		// skip root layout
		{
			template: Associated(
				"sub/index",
				"sub/layout",
				Map{
					"Data":     2,
					"Children": Tmpl("sub/index", 3),
				},
			),
			expected: "<h2>2</h2><p>3</p>",
		},
		{
			template: SubLayoutPage{
				Data:     2,
				Children: SubIndexPage(3),
			},
			expected: "<h2>2</h2><p>3</p>",
		},
		// skip sub layout
		{
			template: Associated(
				"sub/index",
				"layout",
				Map{
					"Data":     1,
					"Children": Tmpl("sub/index", 3),
				},
			),
			expected: "<h1>1</h1><p>3</p>",
		},
		{
			template: LayoutPage{
				Data: 1,
				Children: Children{
					SubIndexPage(3),
				},
			},
			expected: "<h1>1</h1><p>3</p>",
		},
		{
			template: Wrap(&LayoutPage{Data: 1}, SubIndexPage(3)),
			expected: "<h1>1</h1><p>3</p>",
		},
	}
	for _, test := range tests {
		err := templates.Render(buf, test.template)
		if err != nil {
			fmt.Println("err:", err)
			t.Error(err)
		}
		if buf.String() != test.expected {
			t.Errorf("expected: %q, got: %q", test.expected, buf.String())
		}
		buf.Reset()
	}
}
