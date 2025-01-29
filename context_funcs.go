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
			tpl := tp.Tmpl().(tmpl)
			var buf bytes.Buffer
			err := t.ExecuteTemplate(&buf, tpl.name, tpl.data)
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
