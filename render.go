package tmpl

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"sync"
)

// WriteError wraps an error returned after template rendering begins. The
// destination may already contain output, so callers should not write fallback
// content.
type WriteError struct{ error }

func (err *WriteError) Unwrap() error { return err.error }

// IsWriteError reports whether err occurred after template rendering began.
func IsWriteError(err error) bool {
	var writeErr *WriteError
	return errors.As(err, &writeErr)
}

// Render executes tp synchronously. Async values block until they resolve.
func (t *Templates) Render(w io.Writer, tp Template) error {
	return t.render(context.Background(), w, tp, false)
}

// Stream executes tp and streams resolved async templates.
// ctx stops waiting for unresolved values when it is cancelled.
func (t *Templates) Stream(ctx context.Context, w io.Writer, tp Template) error {
	return t.render(ctx, w, tp, true)
}

func (t *Templates) render(ctx context.Context, w io.Writer, tp Template, streaming bool) error {
	node := normalize(tp)
	base, _, _ := node.tmplnode()
	parsed := t.parsed[base]
	if parsed == nil {
		parsed = t.parsed["<root>"]
	}

	session := newSession(ctx, w, streaming)
	funcs, pool, err := t.checkout(parsed)
	if err != nil {
		return err
	}
	funcs.session = session
	defer func() {
		session.cancel()
		funcs.session = nil
		pool.Put(funcs)
	}()

	if err := session.render(funcs.template, node); err != nil {
		return &WriteError{err}
	}
	if !streaming {
		return nil
	}
	if err := session.await(funcs.template); err != nil {
		return &WriteError{err}
	}
	return nil
}

func (t *Templates) checkout(parsed *template.Template) (*contextFuncs, *sync.Pool, error) {
	poolValue, ok := t.pools.Load(parsed)
	if !ok {
		poolValue, _ = t.pools.LoadOrStore(parsed, &sync.Pool{})
	}
	pool := poolValue.(*sync.Pool)
	if funcs := pool.Get(); funcs != nil {
		return funcs.(*contextFuncs), pool, nil
	}
	clone, err := parsed.Clone()
	if err != nil {
		return nil, pool, err
	}
	funcs := &contextFuncs{template: clone}
	clone.Funcs(funcs.funcMap())
	return funcs, pool, nil
}

// session belongs to exactly one Templates.Render or Templates.Stream call.
// Template execution and pending are only touched by the rendering goroutine.
type session struct {
	ctx       context.Context
	cancelFn  context.CancelFunc
	w         io.Writer
	streaming bool
	ch        chan streamTemplate
	pending   uint32
	cid       uint32
	children  []tmplnode
}

func newSession(ctx context.Context, w io.Writer, streaming bool) *session {
	return &session{ctx: ctx, w: w, streaming: streaming}
}

// startPending initializes cancellation and delivery only if an unresolved
// stream needs a waiter goroutine.
func (s *session) startPending() {
	if s.cancelFn != nil {
		return
	}
	s.ctx, s.cancelFn = context.WithCancel(s.ctx)
	s.ch = make(chan streamTemplate)
}

func (s *session) cancel() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
}

func (s *session) nextCID() uint32 {
	s.cid++
	return s.cid
}

func (s *session) flush() {
	if f, ok := s.w.(http.Flusher); ok {
		f.Flush()
	}
}

func (s *session) render(t *template.Template, node tmplnode) error {
	switch n := node.(type) {
	case tmpl:
		return t.ExecuteTemplate(s.w, n.name, n.data)
	case tmplwrap:
		_, name, data := n.tmplnode()
		child, ok := n.child.(tmplnode)
		if !ok {
			return fmt.Errorf("tmpl: invalid child node")
		}
		s.children = append(s.children, child)
		defer func() { s.children = s.children[:len(s.children)-1] }()
		return t.ExecuteTemplate(s.w, name, data)
	}
	return fmt.Errorf("tmpl: invalid render node")
}

func (s *session) renderChildren(t *template.Template) error {
	if len(s.children) == 0 {
		return fmt.Errorf("tmpl: children called outside a layout")
	}
	return s.render(t, s.children[len(s.children)-1])
}
