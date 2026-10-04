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

func streamingTemplates(t *testing.T, source string) *Templates {
	t.Helper()
	return New(fstest.MapFS{"page.html": {Data: []byte(source)}}).Load("page").MustParse()
}

func TestStreamReplacesCommentBoundaries(t *testing.T) {
	templates := streamingTemplates(t, `before{{ stream "value" . }}after{{ define "value" }}<strong>{{ . }}</strong>{{ end }}{{ define "value:pending" }}<em>Loading</em>{{ end }}`)
	value := NewAsync[string, error]()
	var output bytes.Buffer
	w := &resolveOnMarker{Writer: &output, marker: "tmpl:start:", resolve: func() { value.Ok("done") }}

	if err := templates.Stream(context.Background(), w, Tmpl("page", value)); err != nil {
		t.Fatal(err)
	}

	got := output.String()
	if !strings.Contains(got, `<!--tmpl:start:1--><em>Loading</em><!--tmpl:end:1-->`) {
		t.Fatalf("missing comment boundaries: %q", got)
	}
	if strings.Contains(got, `<div data-tmpl-cid=`) {
		t.Fatalf("stream placeholder still uses a div: %q", got)
	}
	if !strings.Contains(got, `<template data-tmpl-cid="1"><strong>done</strong></template>`) {
		t.Fatalf("missing resolved fragment: %q", got)
	}
}

func TestStreamInsideRender(t *testing.T) {
	templates := streamingTemplates(t, `{{ render .Child }}{{ define "child" }}<section>{{ stream "value" . }}</section>{{ end }}{{ define "value" }}<strong>{{ . }}</strong>{{ end }}`)
	value := NewAsync[string, error]()
	var output bytes.Buffer
	w := &resolveOnMarker{Writer: &output, marker: "tmpl:start:", resolve: func() { value.Ok("done") }}
	page := Map{"Child": Tmpl("child", value)}

	if err := templates.Stream(context.Background(), w, Tmpl("page", page)); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, `<section><!--tmpl:start:1--><!--tmpl:end:1--></section>`) {
		t.Fatalf("missing rendered stream placeholder: %q", got)
	}
	if !strings.Contains(got, `<template data-tmpl-cid="1"><strong>done</strong></template>`) {
		t.Fatalf("missing rendered resolved fragment: %q", got)
	}
}

func TestStreamInsideChildren(t *testing.T) {
	templates := streamingTemplates(t, `{{ define "layout" }}before{{ children }}after{{ end }}{{ define "page" }}{{ stream "value" . }}{{ end }}{{ define "value" }}<strong>{{ . }}</strong>{{ end }}`)
	value := NewAsync[string, error]()
	var output bytes.Buffer
	w := &resolveOnMarker{Writer: &output, marker: "tmpl:start:", resolve: func() { value.Ok("done") }}
	page := Wrap(Tmpl("layout", nil), Tmpl("page", value))

	if err := templates.Stream(context.Background(), w, page); err != nil {
		t.Fatal(err)
	}
	got := output.String()
	if !strings.Contains(got, `before<!--tmpl:start:1--><!--tmpl:end:1-->after`) {
		t.Fatalf("missing streamed children placeholder: %q", got)
	}
	if !strings.Contains(got, `<template data-tmpl-cid="1"><strong>done</strong></template>`) {
		t.Fatalf("missing streamed children fragment: %q", got)
	}
}

func TestStreamCancellationStopsWaiters(t *testing.T) {
	templates := streamingTemplates(t, `{{ stream "value" . }}`)
	value := &blockingAsync{started: make(chan struct{}), stopped: make(chan struct{})}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	w := &resolveOnMarker{Writer: io.Discard, marker: "tmpl:start:", resolve: cancel}

	err := templates.Stream(ctx, w, Tmpl("page", value))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	select {
	case <-value.started:
	case <-time.After(time.Second):
		t.Fatal("waiter did not start")
	}
	select {
	case <-value.stopped:
	case <-time.After(time.Second):
		t.Fatal("waiter did not stop after cancellation")
	}
}

func TestInitialRenderErrorStopsWaiters(t *testing.T) {
	templates := streamingTemplates(t, `{{ stream "value" .Value }}{{ .Missing }}`)
	value := &blockingAsync{started: make(chan struct{}), stopped: make(chan struct{})}
	page := struct{ Value asyncValue }{Value: value}

	if err := templates.Stream(context.Background(), io.Discard, Tmpl("page", page)); err == nil {
		t.Fatal("expected initial template error")
	}
	select {
	case <-value.started:
	case <-time.After(time.Second):
		t.Fatal("waiter did not start")
	}
	select {
	case <-value.stopped:
	case <-time.After(time.Second):
		t.Fatal("waiter did not stop after render error")
	}
}

func TestStreamWriteErrorStopsWaiters(t *testing.T) {
	templates := streamingTemplates(t, `{{ stream "value" . }}`)
	value := &blockingAsync{started: make(chan struct{}), stopped: make(chan struct{})}
	err := templates.Stream(context.Background(), failOnScriptWriter{}, Tmpl("page", value))
	if err == nil {
		t.Fatal("expected write error")
	}
	select {
	case <-value.started:
	case <-time.After(time.Second):
		t.Fatal("waiter did not start")
	}
	select {
	case <-value.stopped:
	case <-time.After(time.Second):
		t.Fatal("waiter did not stop after write error")
	}
}

func TestPresentFallbackTemplateErrorsPropagate(t *testing.T) {
	templates := streamingTemplates(t, `{{ stream "value" . }}{{ define "value:pending" }}{{ index . 0 }}{{ end }}`)
	value := NewAsync[string, error]()
	if err := templates.Stream(context.Background(), io.Discard, Tmpl("page", value)); err == nil {
		t.Fatal("expected pending template error")
	}

	templates = streamingTemplates(t, `{{ stream "value" . }}{{ define "value:error" }}{{ index . 0 }}{{ end }}`)
	value = NewAsync[string, error]()
	value.Err(errors.New("failed"))
	if err := templates.Render(io.Discard, Tmpl("page", value)); err == nil {
		t.Fatal("expected error template error")
	}
}

func TestAsyncConcurrentCachedReads(t *testing.T) {
	value := NewAsync[int, error]().(*async[int, error])
	start := make(chan struct{})
	var readers sync.WaitGroup
	for range 8 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			<-start
			for range 10_000 {
				value.getStored()
			}
		}()
	}
	close(start)
	value.Ok(1)
	readers.Wait()
	data, ok := value.getStored()
	if !ok || !data.ok || data.data != 1 {
		t.Fatalf("unexpected cached value: %#v, %v", data, ok)
	}
}

func TestAsyncRejectsDuplicateResolution(t *testing.T) {
	value := NewAsync[int, error]()
	value.Ok(1)
	defer func() {
		if recover() == nil {
			t.Fatal("expected duplicate resolution panic")
		}
	}()
	value.Err(errors.New("too late"))
}

func TestGo(t *testing.T) {
	value := Go(func(value Async[int, error]) { value.Ok(7) })
	data, resolved := value.(asyncValue).get(context.Background())
	if !resolved || !data.ok || data.data != 7 {
		t.Fatalf("unexpected result: %#v, %v", data, resolved)
	}
}

func TestStreamConcurrent(t *testing.T) {
	templates := streamingTemplates(t, `{{ stream "value" . }}{{ define "value" }}{{ . }}{{ end }}`)
	const renders = 32
	var wg sync.WaitGroup
	errs := make(chan error, renders*2)
	for i := range renders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			value := NewAsync[int, error]()
			value.Ok(i)
			var output bytes.Buffer
			errs <- templates.Stream(context.Background(), &output, Tmpl("page", value))
			if output.String() != fmt.Sprint(i) {
				errs <- fmt.Errorf("unexpected output %q", output.String())
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

type resolveOnMarker struct {
	io.Writer
	marker  string
	once    sync.Once
	resolve func()
}

func (w *resolveOnMarker) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte(w.marker)) {
		w.once.Do(w.resolve)
	}
	return w.Writer.Write(p)
}

type blockingAsync struct {
	started chan struct{}
	stopped chan struct{}
}

func (a *blockingAsync) get(ctx context.Context) (streamData, bool) {
	close(a.started)
	<-ctx.Done()
	close(a.stopped)
	return streamData{}, false
}

func (a *blockingAsync) getStored() (streamData, bool) { return streamData{}, false }

type failOnScriptWriter struct{}

func (failOnScriptWriter) Write(p []byte) (int, error) {
	if bytes.Contains(p, []byte(`function tmplComment`)) {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}
