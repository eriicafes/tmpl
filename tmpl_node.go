package tmpl

// tmpl represents a template definition.
type tmpl struct {
	base string
	name string
	data any
}

func (t tmpl) Tmpl() Template { return t }
func (t tmpl) tmplnode() (base, name string, data any) {
	return t.base, t.name, t.data
}

// tmplwrap represents a relationship between a layout and its child template.
type tmplwrap struct {
	base   string
	name   string
	data   any
	layout Template
	child  Template
}

func (w tmplwrap) Tmpl() Template { return w }
func (w tmplwrap) tmplnode() (base, name string, data any) {
	return w.base, w.name, w.data
}

// tmplnode is the normalized internal representation of a Template.
type tmplnode interface {
	Template
	tmplnode() (base, name string, data any)
}

func normalize(tp Template) tmplnode {
	switch n := tp.(type) {
	case tmpl:
		return n
	case tmplwrap:
		layout := normalize(n.layout)
		child := normalize(n.child)
		if outer, ok := layout.(tmplwrap); ok {
			return normalize(tmplwrap{
				layout: outer.layout,
				child: tmplwrap{
					layout: outer.child,
					child:  child,
				},
			})
		}
		base, _, _ := child.tmplnode()
		_, name, data := layout.tmplnode()
		return tmplwrap{
			base:   base,
			name:   name,
			data:   data,
			layout: layout,
			child:  child,
		}
	default:
		return normalize(tp.Tmpl())
	}
}
