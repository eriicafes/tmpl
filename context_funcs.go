package tmpl

import (
	"bytes"
	"fmt"
	"html/template"
)

func contextFuncMap(t *template.Template) template.FuncMap {
	return template.FuncMap{
		"slot":   slotFunc(t),
		"stream": streamFunc(t),
	}
}

func slotFunc(t *template.Template) any {
	return func(data any) (any, error) {
		if str, ok := data.(string); ok {
			return str, nil
		}
		if tp, ok := data.(Template); ok {
			name, data := tp.Template()
			var buf bytes.Buffer
			err := t.ExecuteTemplate(&buf, name, data)
			return template.HTML(buf.String()), err
		}
		return nil, fmt.Errorf("expected a valid slotted content got %T", data)
	}
}

func streamFunc(t *template.Template) any {
	return func(name string, av asyncValuer) (template.HTML, error) {
		return stream(t, name, av)
	}
}
