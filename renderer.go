package tmpl

import (
	"io"
)

// Renderer executes templates.
type Renderer interface {
	// Render executes the template tp and writes the output to w.
	Render(w io.Writer, tp Template) error
	// RenderAssociated executes the associated template atp and writes the output to w.
	RenderAssociated(w io.Writer, atp AssociatedTemplate) error
}

// SyncRenderer blocks on async values. SyncRenderer is not concurrent safe.
func (t Templates) SyncRenderer() Renderer {
	return &renderer{Templates: t}
}

// StreamRenderer streams in templates with async values. StreamRenderer is not concurrent safe.
func (t Templates) StreamRenderer() Renderer {
	return &renderer{Templates: t, stream: newStreamController()}
}

type renderer struct {
	Templates
	w      io.Writer
	stream *streamController
}

// Render executes the template tp and writes the output to w.
// Render uses a SyncRenderer and blocks on async values.
func (t Templates) Render(w io.Writer, tp Template) error {
	return t.SyncRenderer().Render(w, tp)
}

// RenderAssociated executes the associated template for the template tp and writes the output to w.
// RenderAssociated uses a SyncRenderer and blocks on async values.
func (t Templates) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	return t.SyncRenderer().RenderAssociated(w, atp)
}

func (r *renderer) Render(w io.Writer, tp Template) error {
	name, data := tp.Template()
	t := r.Templates[name]
	if t == nil {
		t = r.Templates["<root>"]
	}
	// attach writer to renderer
	r.w = w
	err := t.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return awaitStream(w, t, r.stream)
}

func (r *renderer) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	tname, name, data := atp.AssociatedTemplate()
	t := r.Templates[tname]
	if t == nil {
		t = r.Templates["<root>"]
	}
	// attach writer to renderer
	r.w = w
	err := t.ExecuteTemplate(w, name, data)
	if err != nil || r.stream == nil {
		return err
	}
	return awaitStream(w, t, r.stream)
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
