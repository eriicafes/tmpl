package tmpl

import (
	"maps"
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
