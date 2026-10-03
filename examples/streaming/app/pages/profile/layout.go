package profile_pages

import (
	"tmpl-example/app/pages"

	"github.com/eriicafes/tmpl"
)

type Layout struct {
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	parent := pages.Layout{Title: l.Title}
	return tmpl.Wrap(parent, tmpl.Tmpl("pages/profile/layout", l))
}
