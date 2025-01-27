package tmpl

import (
	"fmt"
	"html/template"
)

var funcMap = template.FuncMap{
	"map":    mapFunc,
	"clsx":   clsx,
	"stream": stream,
}

func mapFunc(v ...any) (map[string]any, error) {
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
}

func clsx(values ...any) (string, error) {
	var result string
	var matching, cond bool
	appendStr := func(s string) {
		if result == "" {
			result = s
		} else {
			result = fmt.Sprintf("%s %s", result, s)
		}
	}
	for _, value := range values {
		if v, ok := value.(bool); ok {
			if matching {
				return "", fmt.Errorf("expected a string after match condition")
			}
			// start matching
			matching, cond = true, v
			continue
		}
		if v, ok := value.(string); ok {
			if matching {
				if cond {
					appendStr(v)
				}
				// reset matching
				matching, cond = false, false
			} else {
				appendStr(v)
			}
			continue
		}
		if value == nil {
			// reset matching
			matching, cond = false, false
			continue
		}
		return "", fmt.Errorf("value must be string or bool")
	}
	if matching {
		return "", fmt.Errorf("expected a string after match condition")
	}
	return result, nil
}
