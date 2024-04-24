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

// Combine composes layout data and the final template data.
//
// Layout templates receive .Data and .Child as template data.
// data may implement interface{ Data() any } to modify the value that gets passed to the template.
func Combine(name string, data ...any) Template {
	var td any
	for i := len(data) - 1; i >= 0; i-- {
		d := data[i]

		// optionally transform template data
		if dd, ok := d.(interface{ Data() any }); ok {
			d = dd.Data()
		}

		if td == nil {
			td = d
		} else {
			td = Map{"Data": d, "Child": td}
		}
	}
	return NewTemplate(name, td)
}
