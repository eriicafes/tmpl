package tmpl

import (
	"bytes"
	"fmt"
	"html/template"
)

// contextFuncs is private to one checked-out execution template. Its
// function closures are installed once when that execution template is made;
// the renderer only assigns session while it has exclusive use of the clone.
type contextFuncs struct {
	template *template.Template
	session  *session
}

func (r *contextFuncs) funcMap() template.FuncMap {
	return template.FuncMap{
		"render":   r.render,
		"stream":   r.stream,
		"await":    r.await,
		"tryawait": r.tryawait,
		"children": r.children,
	}
}

func (r *contextFuncs) render(data any) (any, error) {
	if str, ok := data.(string); ok {
		return str, nil
	}
	if tp, ok := data.(Template); ok {
		node := normalize(tp)
		if s, ok := node.(streamtmpl); ok {
			return r.stream(s.name, s.av)
		}
		leaf, ok := node.(tmpl)
		if !ok {
			return nil, fmt.Errorf("render cannot render a wrapped template")
		}
		var buf bytes.Buffer
		err := r.template.ExecuteTemplate(&buf, leaf.name, leaf.data)
		return template.HTML(buf.String()), err
	}
	return nil, fmt.Errorf("render expects a string or template, got %T", data)
}

func (r *contextFuncs) children() (template.HTML, error) {
	if r.session == nil {
		return "", fmt.Errorf("children called outside a render session")
	}
	return "", r.session.renderChildren(r.template)
}

func (r *contextFuncs) stream(name string, av asyncValue) (template.HTML, error) {
	if av == nil {
		return "", fmt.Errorf("Async is nil")
	}
	if r.session == nil {
		return "", fmt.Errorf("stream called outside a render session")
	}
	if data, stored := av.getStored(); stored {
		return renderSync(r.template, name, data)
	}
	if !r.session.streaming {
		r.session.flush()
		data, resolved := av.get(r.session.ctx)
		if !resolved {
			return "", r.session.ctx.Err()
		}
		return renderSync(r.template, name, data)
	}
	return renderStream(r.template, r.session, name, av)
}

func (r *contextFuncs) await(av asyncValue) (any, error) {
	result, err := r.tryawait(av)
	if err != nil {
		return nil, err
	}
	if !result.Ok {
		if err, ok := result.Err.(error); ok {
			return nil, fmt.Errorf("await: %w", err)
		}
		return nil, fmt.Errorf("await: %v", result.Err)
	}
	return result.Data, nil
}

// awaitresult is the outcome of an Async, as returned by the tryawait func.
// Data holds the value when Ok, and Err holds the error otherwise.
type awaitresult struct {
	Ok   bool
	Data any
	Err  any
}

func (r *contextFuncs) tryawait(av asyncValue) (awaitresult, error) {
	if av == nil {
		return awaitresult{}, fmt.Errorf("Async is nil")
	}
	if r.session == nil {
		return awaitresult{}, fmt.Errorf("await called outside a render session")
	}
	data, stored := av.getStored()
	if !stored {
		r.session.flush()
		var resolved bool
		data, resolved = av.get(r.session.ctx)
		if !resolved {
			return awaitresult{}, r.session.ctx.Err()
		}
	}
	if data.ok {
		return awaitresult{Ok: true, Data: data.data}, nil
	}
	return awaitresult{Err: data.data}, nil
}
