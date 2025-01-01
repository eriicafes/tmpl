package tmpl

// Map is a map[string]any to provide data to templates.
type Map map[string]any

// Template returns a template name and template data.
// The returned template data is directly available to the template.
type Template interface {
	Template() (name string, data any)
}

// AssociatedTemplate returns a base template name, an associated template name and template data.
// The returned template data is directly available to the associated template.
//
// An associated template is any template that is parsed or defined while loading a template.
// This includes all define blocks, autoloads and the loaded template itself.
type AssociatedTemplate interface {
	AssociatedTemplate() (base string, name string, data any)
}

type tp struct {
	name string
	data any
}

func (t tp) Template() (string, any) {
	return t.name, t.data
}

// Tmpl returns a Template with the given template name and data.
//
// When more than one data argument is provided Tmpl composes the arguments as nested template data.
//
// for example the template below
//
//	tp1 := tmpl.Tmpl("template", 1)
//
// will have the following template data
//
//	data1 := 1
//
// while the template below
//
//	tp2 := tmpl.Tmpl("template", 1, "one", true)
//
// will have the following template data
//
//	data2 := tmpl.Map{
//	  "Data": 1,
//	  "Child": tmpl.Map{
//	    "Data":  "one",
//	    "Child": true,
//	  }
//	}
//
// Multiple arguments for Tmpl should be used when returning data for templates that have layouts.
//
// data may implement interface{ Data() any } to modify the value that gets passed to the template.
func Tmpl(name string, data ...any) Template {
	var td any
	for i := len(data) - 1; i >= 0; i-- {
		d := data[i]
		// optionally transform template data
		if dd, ok := d.(interface{ Data() any }); ok {
			d = dd.Data()
		}
		if i == len(data)-1 {
			td = d
		} else {
			td = Map{"Data": d, "Child": td}
		}
	}
	return tp{name, td}
}

type associatedTp struct {
	base string
	name string
	data any
}

func (t associatedTp) AssociatedTemplate() (string, string, any) {
	return t.base, t.name, t.data
}

// AssociatedTmpl returns an associated template with the given template name and template data from a base template.
func AssociatedTmpl(base string, name string, data any) AssociatedTemplate {
	return associatedTp{base, name, data}
}
