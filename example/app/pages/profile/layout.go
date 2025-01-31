package profile_pages

import (
	"github.com/eriicafes/tmpl"
	"github.com/eriicafes/tmpl/example/app/pages"
)

type Layout struct {
	tmpl.Children
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	parent := pages.Layout{Title: l.Title}
	return tmpl.Wrap(&parent, tmpl.Associated(l.Base(), "pages/profile/layout", l))
}
