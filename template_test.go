package tmpl

import (
	"reflect"
	"testing"
)

func TestTemplateWithSingleData(t *testing.T) {
	tp := Tmpl("template", 1)
	name, data := tp.Template()
	expectedName := "template"
	expectedData := 1
	if name != expectedName {
		t.Errorf("name expected: %q got: %q\n", expectedName, name)
	}
	if data != expectedData {
		t.Errorf("data expected: %#v got: %#v\n", expectedData, data)
	}
}

func TestTemplateWithNestedData(t *testing.T) {
	tp := Tmpl("template", 1, "one", true)
	name, data := tp.Template()
	expectedName := "template"
	expectedData := Map{
		"Data": 1,
		"Child": Map{
			"Data":  "one",
			"Child": true,
		},
	}
	if name != expectedName {
		t.Errorf("name expected: %q got: %q\n", expectedName, name)
	}
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("data expected: %#v got: %#v\n", expectedData, data)
	}
}
