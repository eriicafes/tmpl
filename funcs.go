package tmpl

import (
	"fmt"
	"html/template"
)

var funcMap = template.FuncMap{
	"map": func(v ...any) (map[string]any, error) {
		if len(v)%2 != 0 {
			return nil, fmt.Errorf("key %v missing value", v[len(v)-1])
		}
		m := make(map[string]any, len(v)/2)

		for i := 0; i < len(v); i += 2 {
			if k, ok := v[i].(string); ok {
				m[k] = v[i+1]
			} else {
				return nil, fmt.Errorf("expected string key found %T", v[i])
			}
		}

		return m, nil
	},

	"stream": stream,
}
