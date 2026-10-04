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
		"children": r.children,
	}
}

func (r *contextFuncs) render(data any) (any, error) {
	if str, ok := data.(string); ok {
		return str, nil
	}
	if tp, ok := data.(Template); ok {
		node := normalize(tp)
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
	return stream(r.template, r.session, name, av)
}
