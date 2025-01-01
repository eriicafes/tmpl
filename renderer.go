package tmpl

import (
	"html/template"
	"io"
)

// Renderer executes templates.
type Renderer interface {
	// Render executes the Template tp and writes the output to w.
	Render(w io.Writer, tp Template) error
	// RenderAssociated executes the associated template for the Template tp and writes the output to w.
	RenderAssociated(w io.Writer, atp AssociatedTemplate) error
	// Unwrap returns the underlying renderer.
	Unwrap() *renderer
}

type renderer struct {
	t      Templates
	cache  streamCache
	stream *streamController
	tp     *template.Template
}

// Renderer blocks on async values. It should not be used concurrently.
func (t Templates) Renderer() Renderer {
	return &renderer{t, make(streamCache), nil, nil}
}

// StreamRenderer streams in templates with async values. It should not be used concurrently.
func (t Templates) StreamRenderer() Renderer {
	return &renderer{t, make(streamCache), newStreamController(), nil}
}

func (r *renderer) Render(w io.Writer, tp Template) error {
	name, data := tp.Template()
	tmpl := r.t[name]
	if tmpl == nil {
		tmpl = r.t["<root>"]
	}
	// attach template to renderer
	r.tp = tmpl
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return awaitStream(r, w)
}

func (r *renderer) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	tname, name, data := atp.AssociatedTemplate()
	tmpl := r.t[tname]
	if tmpl == nil {
		tmpl = r.t["<root>"]
	}
	// attach template to renderer
	r.tp = tmpl
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return awaitStream(r, w)
}

func (r *renderer) Unwrap() *renderer {
	return r
}
