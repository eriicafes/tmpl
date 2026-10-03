package pages

import "github.com/eriicafes/tmpl"

type Layout struct {
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	return tmpl.Tmpl("pages/layout", l)
}
