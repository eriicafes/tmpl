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

func init() {
	tmpl.Define(`
<main class="max-w-3xl mx-auto px-4 py-8 flex flex-col items-center justify-center gap-8">
    <div class="flex flex-col items-center">
        <h1 class="text-2xl sm:text-3xl font-medium">{{ .Greeting }}</h1>
        <a href="/profile" class="text-xs underline px-2 py-1">
            Click to update
        </a>
    </div>

    <span id="count" class="size-10 flex items-center justify-center rounded-full bg-zinc-100 text-zinc-700 dark:text-zinc-700">
        {{ .Count }}
    </span>

    {{ template "components/button" map 
        "id" "counter" 
        "content" "Increment count"
        "attrs" "data-theme-toggle"
    }}
</main>

{{ define "script" }}
    {{ vite_script "app/pages/index.ts" }}
{{ end }}
`)
}
