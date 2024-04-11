package tmpl

type layoutData struct {
	Data  any
	Child Template
}

func (ld layoutData) Template() (string, any) {
	name, _ := ld.Child.Template()
	return name, ld
}

// Combine combines layout templates and the final template.
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
		tmpl = layoutData{Data: d, Child: tmpl}
	}
	return tmpl
}
