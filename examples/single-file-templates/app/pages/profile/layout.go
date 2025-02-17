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

func init() {
	tmpl.Define(`
<main class="w-full max-w-sm mx-auto px-4 py-8 flex flex-col justify-center gap-8">
    {{ slot .Children }}
</main>

{{ define "script" }}
    <script>
        console.log("In profile layout")
    </script>

    {{ block "sub/script" . }}
        <script>
            console.log("Should be overriden")
        </script>
    {{ end }}
{{ end }}
`)
}
