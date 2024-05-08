package tmpl

import (
	"reflect"
	"testing"
)

func TestMake(t *testing.T) {
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
		t.Errorf("name: expected %s got %s\n", expectedName, name)
	}
	if !reflect.DeepEqual(data, expectedData) {
		t.Errorf("data: expected %#v got %#v\n", expectedData, data)
	}
}
