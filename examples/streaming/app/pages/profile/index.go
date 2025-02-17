package profile_pages

import "github.com/eriicafes/tmpl"

type Index struct {
	Title string
	Name  string
}

func (i Index) Tmpl() tmpl.Template {
	parent := Layout{Title: i.Title}
	return tmpl.Wrap(&parent, tmpl.Tmpl("pages/profile/index", i))
}
