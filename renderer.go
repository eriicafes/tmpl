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
}

type renderer struct {
	t      Templates
	stream *streamController
	tp     *template.Template
	w      io.Writer
}

// Render executes the Template tp and writes the output to w.
// Render uses a SyncRenderer.
func (t Templates) Render(w io.Writer, tp Template) error {
	return t.SyncRenderer().Render(w, tp)
}

// RenderAssociated executes the associated template for the Template tp and writes the output to w.
// RenderAssociated uses a SyncRenderer.
func (t Templates) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	return t.SyncRenderer().RenderAssociated(w, atp)
}

// SyncRenderer blocks on async values. SyncRenderer is not concurrent safe.
func (t Templates) SyncRenderer() Renderer {
	return &renderer{t, nil, nil, nil}
}

// StreamRenderer streams in templates with async values. StreamRenderer is not concurrent safe.
func (t Templates) StreamRenderer() Renderer {
	return &renderer{t, newStreamController(), nil, nil}
}

func (r *renderer) Render(w io.Writer, tp Template) error {
	name, data := tp.Template()
	tmpl := r.t[name]
	if tmpl == nil {
		tmpl = r.t["<root>"]
	}
	// attach template and writer to renderer
	r.tp, r.w = tmpl, w
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return r.awaitStream()
}

func (r *renderer) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	tname, name, data := atp.AssociatedTemplate()
	tmpl := r.t[tname]
	if tmpl == nil {
		tmpl = r.t["<root>"]
	}
	// attach template and writer to renderer
	r.tp, r.w = tmpl, w
	err := tmpl.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return r.awaitStream()
}

func (r *renderer) Unwrap() Renderer {
	return r
}

func getRenderer(r Renderer) *renderer {
	switch rr := r.(type) {
	case *renderer:
		return rr
	case interface{ Unwrap() Renderer }:
		return getRenderer(rr.Unwrap())
	default:
		return nil
	}
}
