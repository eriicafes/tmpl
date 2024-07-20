# Go Tmpl

### Simple and powerful go templates API with zero code generation.

Tmpl makes go templates composable, easy to work with and predictable, suitable for rendering pages with layouts or rendering page partials with frameworks like [htmx.org](https://htmx.org).

Tmpl only organizes the way you load [go templates](https://pkg.go.dev/html/template).

## Installation

```sh
go get github.com/eriicafes/tmpl
```

## Features

- Render templates with structs
- Colocate template and template data
- Automatic templates loading
- Supports template layouts
- Zero code generation required
- Pure [go templates](https://pkg.go.dev/html/template) (zero dependencies)

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
    SetLayoutRoot("pages").
    OnLoad(func(name string, t *template.Template) {
        // called on template load, before template is parsed
        // register template funcs here
        t.Funcs(template.FuncMap{})
    }).
    MustParse()
```

## Autoload templates

Autoloaded templates are available in all loaded templates.

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
    LoadWithLayouts("pages").
    MustParse()
```

### Load single template with any associated templates.

The loaded template is named after the last template argument.
All preceeding template arguments are associated with the loaded template.

```go
tp := tmpl.New("templates").
    Load("partials/header", "partials/footer", "pages/index").
    MustParse()

// in the above example, the loaded template is named "pages/index"
// however "partials/header" and "partials/footer" are available within "pages/index"
```

## Render templates

### Render template with struct (recommended)

A template is any type that implements `tmpl.Template`. The Template method returns the template name and template data.

> The rendered template name and the template data are tightly coupled.

```go
type Home struct {
    Title string
}

func (h Home) Template() (string, any) {
    return "pages/home", h
}

func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()

    err := tp.Render(os.Stdout, Home{Title: "Homepage"})
}
```

### Render template with `tmpl.Tmpl`

`tmpl.Tmpl` wraps any value in an internal struct that implements `tmpl.Template`.

> The rendered template name and the template data are loosely coupled.

```go
func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()

    err := tp.Render(os.Stdout, tmpl.Tmpl("pages/home", tmpl.Map{
        "Title": "Homepage 2",
    }))
}
```

### Render template with struct and `tmpl.Tmpl` (recommended for layouts)

Use `tmpl.Tmpl` to compose template data.
When more than one data argument is provided `tmpl.Tmpl` composes the arguments as nested template data.
```go
// for example the template below
tp1 := tmpl.Tmpl("template", 1)
// will have the following template data
data1 = 1

// while the template below
tp2 := tmpl.Tmpl("template", 1, "one", true)
// will have the following template data
data2 := tmpl.Map{
  "Data": 1,
  "Child": tmpl.Map{
    "Data":  "one",
    "Child": true,
  }
}
```

> The rendered template name and the template data are tightly coupled.

```go
type Layout struct {
    Title string
}

type Home struct {
    Layout Layout
    Username string
}

func (h Home) Template() (string, any) {
    return tmpl.Tmpl("pages/home", h.Layout, h).Template()
}

func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()

    err := tp.Render(os.Stdout, Home{
        Layout: Layout{Title: "Homepage"},
        Username: "Johndoe",
    })
}
```

## Layouts

Templates with layouts render their layout template and define a block to fill the layout template's slots.
See the example below:

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

### HTML structure / Data flow

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
  <p>Hello {{ .Username }}</p>
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
    layout := Layout{Title: "Homepage"}
    return tmpl.Tmpl("pages/index", layout, i).Template()
}
```
> When more than one data argument is provided `tmpl.Tmpl` composes the arguments as nested template data.

`templates/pages/index.html` is the entry point. It renders the layout and passes all the template data to the layout. `templates/pages/layout.html` receives the template data which should contain `.Data` and `.Child` fields (`.Data` is the layout data and `.Child` is the data for the child template).

`templates/pages/index.html` renders the layout and defines the `content` block.
`templates/pages/layout.html` renders the html shell and renders the `content` block already defined by `templates/pages/index.html`.

### Render layouts with `tmpl.Tmpl`

```go
func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()

    layout := tmpl.Map{"Title": "Homepage"}
    page := tmpl.Map{"Username": "Johndoe"}
    err := tp.Render(os.Stdout, tmpl.Tmpl("pages/index", layout, page))
}
```

### Render layouts with a custom `tmpl.TmplFunc`
The default field names for the nested template data with `tmpl.Tmpl` are `.Data` and `.Child`. If you want to use different names for them you can use `tmpl.TmplFunc`.

```go
func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()
    customTmpl := tmpl.TmplFunc("Props", "ChildProps")

    layout := tmpl.Map{"Title": "Homepage"}
    page := tmpl.Map{"Username": "Johndoe"}
    err := tp.Render(os.Stdout, customTmpl("pages/index", layout, page))
}
```
> template data is now a map of "Props" and "ChildProps" you should update the template files accordingly.


### Optionally modify data with `tmpl.Tmpl`

The arguments passed to `tmpl.Data` may implement the Data method to modify the value before it is passed to the template.

See example below:

```go
type Layout struct {
    Title string
}

func (l Layout) Data() any {
    // use fallback title if not set
    if l.Title == "" {
        l.Title = "Default title"
    }
    return l
}

type IndexPage struct {
    Username string
}

func (p IndexPage) Template() (string, any) {
    // NOTE: no title passed for layout
    return tmpl.Tmpl("pages/index", Layout{}, p).Template()
}

func main() {
    tp := tmpl.New("templates").LoadWithLayouts("pages").MustParse()

    err := tp.Render(os.Stdout, IndexPage{Username: "Johndoe"})
    // template data would be
    // tmpl.Map{
    //   "Data": Layout{Title: "Default title"},
    //   "Child": IndexPage{Username: "Johndoe"},
    // }
}
```

## Associated Templates

Associated templates are named templates within a loaded template. This is useful for rendering partials defined within the template. Below is an example implementing a counter with htmx.

```html
<!-- templates/pages/counter.html -->
{{ template "pages/layout" . }}

{{ define "counter" }}
<form hx-post="/counter" hx-swap="outerHTML">
    <p>Count: {{ .Count }}</p>
    <button>Increment</button>
</form>
{{ end }}

{{ define "content" }}
<main>
    <p>Counter</p>
    {{ template "counter" .Counter }}
</main>
{{ end }}
```

```go
// templates/pages/counter.go
package pages

import (
    "github.com/eriicafes/tmpl"
)

type Counter struct {
	Count int
}

func (c Counter) AssociatedTemplate() (string, string, any) {
    return "pages/index", "counter", c
}

type CounterPage struct {
    Counter Counter
}

func (p CounterPage) Template() (string, any) {
    return tmpl.Tmpl("pages/index", nil, p).Template()
}
```

```go
// templates/pages/counter.go
package pages

import (
    ...
)

var tp tmpl.Templates

func GetCounterPage(w http.ResponseWriter, r *http.Request) {
    newCounter := pages.Counter{Count: 0}
    // render full page
    tp.Render(w, pages.CounterPage{Counter: newCounter})
}

func PostCounterPage(w http.ResponseWriter, r *http.Request) {
    var prevCount int = ...
    // render page partial only
    tp.RenderAssociated(w, pages.Counter{Count: prevCount+1})

    // or if AssociatedTemplate method is not implemented for Counter
    tp.RenderAssociated(w, tmpl.Associated(
        "pages/index",
        "counter",
        Counter{Count: prevCount + 1}),
    )
}
```

## Funcs

Tmpl predefines some template functions.

### map
Returns a map from successive arguments. Arguments length must be even.
```html
{{ $data := map "key" "value" }}
<div>
    <h1>{{ index $data "key" }}</h1>
</div>

<!-- or use as props when calling another template -->
{{ define "button" }}
<button type="{{ .type }}">{{ .text }}</button>
{{ end }}

{{ template "button" map "text" "Click me!" "type" "submit" }}
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
