package tmpl

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
)

func TestRender(t *testing.T) {
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
			{{- render .errorChildren -}}
			{{ else }}
			{{- render .children -}}
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

func TestChildren(t *testing.T) {
	fs := fstest.MapFS{
		"layout.html": {Data: []byte(`<main>{{ children }}</main>`)},
		"page.html":   {Data: []byte(`<span>{{ . }}</span>`)},
	}
	templates := New(fs).Load("layout", "page").MustParse()

	const renders = 32
	errs := make(chan error, renders)
	var wg sync.WaitGroup
	for i := range renders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var output bytes.Buffer
			page := Wrap(Tmpl("layout", nil), Tmpl("page", i))
			if err := templates.Render(&output, page); err != nil {
				errs <- err
				return
			}
			want := fmt.Sprintf("<main><span>%d</span></main>", i)
			if got := output.String(); got != want {
				errs <- fmt.Errorf("got %q, want %q", got, want)
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
}

func TestChildrenOutsideLayout(t *testing.T) {
	templates := New(fstest.MapFS{
		"page.html": {Data: []byte(`{{ children }}`)},
	}).Load("page").MustParse()

	err := templates.Render(io.Discard, Tmpl("page", nil))
	if err == nil || !strings.Contains(err.Error(), "children called outside a layout") {
		t.Fatalf("expected children error, got %v", err)
	}
}

func TestRenderRejectsComposedLayout(t *testing.T) {
	templates := New(fstest.MapFS{
		"render.html": {Data: []byte(`{{ render . }}`)},
	}).Load("render").MustParse()

	err := templates.Render(io.Discard, Tmpl("render", Wrap(Tmpl("layout", nil), Tmpl("page", nil))))
	if err == nil || !strings.Contains(err.Error(), "render cannot render a wrapped template") {
		t.Fatalf("expected composed render error, got %v", err)
	}
}

func TestRenderRejectsInvalidContent(t *testing.T) {
	templates := New(fstest.MapFS{
		"render.html": {Data: []byte(`{{ render . }}`)},
	}).Load("render").MustParse()

	err := templates.Render(io.Discard, Tmpl("render", 1))
	if err == nil || !strings.Contains(err.Error(), "render expects a string or template, got int") {
		t.Fatalf("expected invalid render content error, got %v", err)
	}
}
