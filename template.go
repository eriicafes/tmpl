package tmpl

// Map is a map[string]any to provide data to templates.
type Map map[string]any

type tp struct {
	name string
	data any
}

func (t tp) Template() (string, any) {
	return t.name, t.data
}

// NewTemplate creates a tmpl.Template with a template name and template data.
func NewTemplate(name string, data any) Template {
	return tp{name, data}
}

type layoutTp struct {
	Data  any
	Child Template
}

func (ld layoutTp) Template() (string, any) {
	name, _ := ld.Child.Template()
	return name, ld
}

// Combine combines layouts and the final template.
// The last argument must be a tmpl.Template.
//
// All arguments except the last are layout data.
// Layout templates receive .Data and .Child as template data.
//
// Layout data may implement the interface{ Data() any } to modify the final value that gets passed to the layout template as .Data.
func Combine(data ...any) Template {
	var tmpl Template
	for i := len(data) - 1; i >= 0; i-- {
		d := data[i]
		if t, ok := d.(Template); ok {
			tmpl = t
			continue
		}
		if dd, ok := d.(interface{ Data() any }); ok {
			d = dd.Data()
		}
		tmpl = layoutTp{Data: d, Child: tmpl}
	}
	return tmpl
}
