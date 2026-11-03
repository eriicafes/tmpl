package tmpl

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/fstest"
	"time"
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

func TestAwait(t *testing.T) {
	templates := streamingTemplates(t, `before{{ template "value" (await .) }}{{ define "value" }}<b>{{ . }}</b>{{ end }}`)
	value := NewAsync[string, error]()
	go func() { time.Sleep(10 * time.Millisecond); value.Ok("done") }()
	for _, streaming := range []bool{false, true} {
		var output bytes.Buffer
		var err error
		if streaming {
			err = templates.Stream(context.Background(), &output, Tmpl("page", value))
		} else {
			err = templates.Render(&output, Tmpl("page", value))
		}
		if err != nil {
			t.Fatal(err)
		}
		if got := output.String(); got != "before<b>done</b>" {
			t.Fatalf("streaming %v: got %q", streaming, got)
		}
	}
}

func TestAwaitAsProp(t *testing.T) {
	templates := streamingTemplates(t, `{{ template "card" props "children" (tmpl "value" (await .)) }}{{ define "card" }}<section>{{ render .children }}</section>{{ end }}{{ define "value" }}{{ . }}{{ end }}`)
	value := NewAsync[string, error]()
	value.Ok("done")
	var output bytes.Buffer
	if err := templates.Render(&output, Tmpl("page", value)); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "<section>done</section>" {
		t.Fatalf("got %q", got)
	}
}

var errAwaitTest = errors.New("value failed")

func TestAwaitError(t *testing.T) {
	templates := streamingTemplates(t, `{{ await . }}`)
	value := NewAsync[string, error]()
	value.Err(errAwaitTest)
	err := templates.Render(io.Discard, Tmpl("page", value))
	if !errors.Is(err, errAwaitTest) {
		t.Fatalf("expected the value's error, got %v", err)
	}

	other := NewAsync[string, string]()
	other.Err("bad input")
	err = templates.Render(io.Discard, Tmpl("page", other))
	if err == nil || !strings.Contains(err.Error(), "await: bad input") {
		t.Fatalf("expected the formatted error, got %v", err)
	}
}

func TestTryAwait(t *testing.T) {
	templates := streamingTemplates(t, `{{ with tryawait . }}{{ if .Ok }}ok:{{ .Data }}{{ else }}err:{{ .Err }}{{ end }}{{ end }}`)
	ok := NewAsync[string, error]()
	ok.Ok("done")
	failed := NewAsync[string, error]()
	failed.Err(errAwaitTest)
	for value, want := range map[Async[string, error]]string{ok: "ok:done", failed: "err:value failed"} {
		var output bytes.Buffer
		if err := templates.Render(&output, Tmpl("page", value)); err != nil {
			t.Fatal(err)
		}
		if got := output.String(); got != want {
			t.Fatalf("got %q, want %q", got, want)
		}
	}
}

func TestAwaitStopsWithContext(t *testing.T) {
	templates := streamingTemplates(t, `{{ await . }}`)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	err := templates.Stream(ctx, io.Discard, Tmpl("page", NewAsync[string, error]()))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected the context's error, got %v", err)
	}
}
