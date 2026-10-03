package tmpl

import (
	"maps"
	"reflect"
	"testing"
)

func TestMap(t *testing.T) {
	tests := []struct {
		input  []any
		output map[string]any
		err    string
	}{
		{
			input:  []any{"one", 1, "two", 2},
			output: map[string]any{"one": 1, "two": 2},
			err:    "",
		},
		{
			input:  []any{"one", 1, "two", 2, "three"},
			output: nil,
			err:    "key three missing value",
		},
		{
			input:  []any{"one", 1, "two", 2, 3, "three"},
			output: nil,
			err:    "expected string key found int",
		},
	}

	for _, test := range tests {
		output, err := mapFunc(test.input...)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("expected err: %q got: %q", test.err, err)
			}
		} else {
			if test.err != "" {
				t.Errorf("expected err: %q got: %v", test.err, err)
			}
		}
		if !maps.Equal(output, test.output) {
			t.Errorf("expected: %q got: %q", test.output, output)
		}
	}
}

func TestProps(t *testing.T) {
	props, ok := funcMap["props"].(func(...any) (map[string]any, error))
	if !ok {
		t.Fatal("props is not a map function")
	}

	tests := []struct {
		input  []any
		output map[string]any
		err    string
	}{
		{
			input:  []any{"class", "primary", "disabled", true},
			output: map[string]any{"class": "primary", "disabled": true},
		},
		{
			input: []any{"class", "primary", "disabled"},
			err:   "key disabled missing value",
		},
		{
			input: []any{"class", "primary", 1, true},
			err:   "expected string key found int",
		},
	}

	for _, test := range tests {
		output, err := props(test.input...)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("expected err: %q got: %q", test.err, err)
			}
		} else if test.err != "" {
			t.Errorf("expected err: %q got: %v", test.err, err)
		}
		if !maps.Equal(output, test.output) {
			t.Errorf("expected: %q got: %q", test.output, output)
		}
	}
}

func TestTmpl(t *testing.T) {
	tmplFunc, ok := funcMap["tmpl"].(func(string, any) Template)
	if !ok {
		t.Fatal("tmpl is not a template function")
	}

	tests := []struct {
		name string
		data any
	}{
		{name: "component", data: "text"},
		{name: "components/button", data: Map{"disabled": true}},
	}
	for _, test := range tests {
		base, name, data := Info(tmplFunc(test.name, test.data))
		if base != test.name || name != test.name || !reflect.DeepEqual(data, test.data) {
			t.Errorf("expected (%q, %q, %#v), got (%q, %q, %#v)", test.name, test.name, test.data, base, name, data)
		}
	}
}

func TestClsx(t *testing.T) {
	tests := []struct {
		input  []any
		output string
		err    string
	}{
		{
			input:  []any{"one", "two", "three"},
			output: "one two three",
			err:    "",
		},
		{
			input:  []any{"one", "two", true, "three"},
			output: "one two three",
			err:    "",
		},
		{
			input:  []any{"one", "two", false, "three"},
			output: "one two",
			err:    "",
		},
		{
			input:  []any{true, "one", false, "two", "three"},
			output: "one three",
			err:    "",
		},
		{
			input:  []any{"one", "two", true, false, "three"},
			output: "",
			err:    "expected a string after match condition",
		},
		{
			input:  []any{"one", "two", "three", true},
			output: "",
			err:    "expected a string after match condition",
		},
		{
			input:  []any{"one", "two", "three", 1},
			output: "",
			err:    "value must be string or bool",
		},
		{
			input:  []any{"one", "two", nil, "three"},
			output: "one two three",
			err:    "",
		},
		{
			// true cond keeps the nil value but skips it as it is nil
			input:  []any{"one", "two", true, nil, "three"},
			output: "one two three",
			err:    "",
		},
		{
			// false cond omits the nil value
			input:  []any{"one", "two", false, nil, "three"},
			output: "one two three",
			err:    "",
		},
	}

	for _, test := range tests {
		output, err := clsxFunc(test.input...)
		if err != nil {
			if err.Error() != test.err {
				t.Errorf("expected err: %q got: %q", test.err, err)
			}
		} else {
			if test.err != "" {
				t.Errorf("expected err: %q got: %v", test.err, err)
			}
		}
		if output != test.output {
			t.Errorf("expected: %q got: %q", test.output, output)
		}
	}
}
