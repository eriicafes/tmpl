package pages

import (
	"fmt"

	"github.com/eriicafes/tmpl"
)

type Index struct {
	Layout
	Name  string
	Count int
}

func (i Index) Tmpl() tmpl.Template {
	return tmpl.Wrap(&i.Layout, tmpl.Tmpl("pages/index", i))
}

func (i Index) Greeting() string {
	if i.Name == "" {
		return "Please update your profile"
	}
	return fmt.Sprintf("Welcome %s", i.Name)
}
