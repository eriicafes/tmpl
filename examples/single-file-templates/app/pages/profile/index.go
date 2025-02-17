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

func init() {
	tmpl.Define(`
<form method="post" class="grid gap-4 p-4">
    <h1>Update your profile</h1>

    <div class="grid gap-1">
        {{ template "components/input" map 
        "name" "name"
        "placeholder" "Enter your name"
        "value" .Name
        }}
        <label class="text-xs opacity-60 px-1">Your name</label>
    </div>

    {{ template "components/button" map "content" "Save" }}
</form>

{{ define "sub/script" }}
    <script>
        console.log("In profile page")
    </script>
{{ end }}
`)
}
