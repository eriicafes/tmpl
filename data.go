package tmpl

// Map is a map[string]any to provide data to templates.
type Map map[string]any

type data struct {
	name string
	data any
}

func (m data) Template() (string, any) {
	return m.name, m.data
}

// Data creates a tmpl.Template with a template name and template data.
func Data(name string, d any) Template {
	return data{name, d}
}
