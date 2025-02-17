package pages

import "github.com/eriicafes/tmpl"

type Layout struct {
	tmpl.Children
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	return tmpl.Associated(l.Base(), "pages/layout", l)
}
