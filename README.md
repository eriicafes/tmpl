# Go Tmpl

### Simple and powerful go templates API with zero build.

Tmpl makes go templates composable, easy to work with and predictable, perfect for rendering pages with layouts or rendering page partials with frameworks like [htmx.org](https://htmx.org).

Tmpl only organizes the way you load [go templates](https://pkg.go.dev/html/template) with zero dependencies.

## Installation

```sh
go get github.com/eriicafes/tmpl
```

## Features

- Execute templates with structs
- Automatic templates loading
- Supports template layouts
- Zero build required
- Pure go templates (zero dependencies)

## Setup

### Initialize templates.

```go
tp, err := tmpl.New("templates").Parse()

// or

tp := tmpl.New("templates").MustParse()
```

### Configure templates (optional)

```go
tp := tmpl.New("templates").
    SetExt("tmpl"). // default is "html"
    SetLayoutFilename("_layout"). // default is "layout"
    SetLayoutDir("pages").
    OnLoad(func(name string, t *template.Template) {
        // called on template load, before template is parsed
        // register template funcs here
        t.Funcs(template.FuncMap{})
    }).
    MustParse()
```

## Autoload templates

Autoloaded templates are available in all templates.

```go
tp := tmpl.New("templates").
    Autoload("components").
    MustParse()
```

Use shallow autoload to skip templates in sub-directories.

```go
tp := tmpl.New("templates").
    AutoloadShallow("components/ui").
    AutoloadShallow("components/icons").
    MustParse()
```

## Load templates

### Load all templates including layouts (recommended).

Load all templates like in a file-based router. By default the layout filename is `layout`.

```go
tp := tmpl.New("templates").
    LoadDir("pages").
    MustParse()
```

### Load single templates with any associated templates.

The loaded template is named after the last template argument.
All preceeding templates arguments are associated with the loaded template.

```go
tp := tmpl.New("templates").
    Load("partials/header", "partials/footer", "pages/index").
    MustParse()

// in the above example, the loaded template is named "pages/index"
// however "partials/header" and "partials/footer" are available within "pages/index"
```

## Render templates

### Render struct template (recommended)

A template is any type that implements `tmpl.Template`. The Template method returns the template name and template data.

Here, the rendered template name and the template data are tightly coupled.

```go
type Home struct {
	Title string
}

func (h Home) Template() (string, any) {
	return "pages/home", h
}

func main() {
	tp := tmpl.New("templates").LoadDir("pages").MustParse()

	err := tp.Render(os.Stdout, Home{Title: "Homepage"})
}
```

### Render template using `tmpl.NewTemplate`

`tmpl.NewTemplate` wraps any value in an internal struct that implements `tmpl.Template`.

Here, the rendered template name and the template data are loosely coupled.

```go
func main() {
	tp := tmpl.New("templates").LoadDir("pages").MustParse()

	err := tp.Render(os.Stdout, tmpl.NewTemplate("pages/home", tmpl.Map{
        "Title": "Homepage 2",
    }))
}
```

## Layouts templates

Templates with layouts are rendered by rendering the parent template and defining a block to fill the template's slot.

### Directory structure

```text
/
├── templates/
│   ├── components/
│   ├── pages/
│   │   │── index.go
│   │   │── index.html
│   │   │── layout.go
│   │   └── layout.html
└── main.go
```

### HTML structure

```html
<!-- templates/pages/layout.html -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{{ .Data.Title }}</title>
  </head>
  <body>
    <header>
      <h1>{{ .Data.Title }}</h1>
    </header>
    {{ template "content" .Child }}
  </body>
</html>
```

```go
// templates/pages/layout.go
package pages

type Layout struct {
    Title string
}
```

```html
<!-- templates/pages/index.html -->
{{ template "pages/layout" . }}
{{ define "content" }}
<main>
  <p>{{ .Username }}</p>
</main>
{{ end }}
```

```go
// templates/pages/index.go
package pages

import (
	"github.com/eriicafes/tmpl"
)

type Index struct {
    Username string
}

func (i Index) Template() (string, any) {
    layout := Layout{Title: "Home title"}
    return "pages/index", tmpl.Combine(layout, i)
}
```

`templates/pages/layout.html` renders the html shell and renders the `content` block. `templates/pages/index.html` renders the layout and defines the `content` block.

### Data flow

`templates/pages/index.html` is the entry point. It renders the layout and passes all the template data to the layout. `templates/pages/layout.html` receives the template data which should contain `.Data` and `.Child` fields (`.Data` is the layout data and `.Child` is the data for the child template).

**Render struct template with layout (recommended)**

```go
func main() {
	tp := tmpl.New("templates").LoadDir("pages").MustParse()

	err := tp.Render(os.Stdout, pages.Index{Username: "Johndoe"}) // just works
}
```

**Render template using `tmpl.NewTemplate`**

```go
func main() {
	tp := tmpl.New("templates").LoadDir("pages").MustParse()

	err := tp.Render(os.Stdout, tmpl.NewTemplate("pages/index", tmpl.Map{
        "Data": tmpl.Map{
            "Title": "Home title 2",
        },
        "Child": tmpl.Map{
            "Username": "Johndoe2",
        },
    }))
}
```

## Clone templates

Clone templates to share similar configurations between templates.

```go
tp1 := tmpl.New("templates").SetExt("tmpl")
// tmpl extension applies to tp1, tp2 and tp3

tp2, err := tp1.Clone()
tp2.Autoload("components/ui")
// components/ui autoload applies only to tp2

// MustClone panics on clone error
tp3 := tp1.MustClone().Autoload("components/icons")
// components/icons autoload applies only to tp3
```
