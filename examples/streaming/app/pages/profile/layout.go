package profile_pages

import (
	"tmpl-example/app/pages"

	"github.com/eriicafes/tmpl"
)

type Layout struct {
	tmpl.Children
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	parent := pages.Layout{Title: l.Title}
	return tmpl.Wrap(&parent, tmpl.Associated(l.Base(), "pages/profile/layout", l))
}
