# Go Tmpl

### Simple and powerful go templates with zero code generation.

Tmpl makes go templates composable, easy to work with and predictable, suitable for rendering pages with layouts or rendering page partials with frameworks like [htmx.org](https://htmx.org).

Tmpl only organizes the way you load [go templates](https://pkg.go.dev/html/template).

Tmpl provides first-party support for [Vite](https://vite.dev).
See [Vite integration](vite/README.md) section.

## Installation

```sh
go get github.com/eriicafes/tmpl
```

## Features

- Render templates with structs
- Colocate template and template data
- Automatic templates loading
- Supports template layouts
- Pure [go templates](https://pkg.go.dev/html/template) (zero dependencies, zero code generation)
- [HTML streaming](html-streaming.md) support
- First-party [Vite integration](vite/README.md)

## Setup

### Initialize templates.

```go
fs := os.DirFS("templates")

tp, err := tmpl.New(fs).Parse()

// or

tp := tmpl.New(fs).MustParse()
```

### Configure templates (optional)

```go
tp := tmpl.New(os.DirFS("templates")).
    SetExt("tmpl"). // default is "html"
    SetLayoutFilename("_layout"). // default is "layout"
    Funcs(funcMaps...). // register template funcs here
    OnLoad(func(name string, t *template.Template) {
        // called on template load, before template is parsed
        t.Funcs(template.FuncMap{})
    }).
    MustParse()
```

## Autoload templates

Autoloaded templates are available in all loaded templates.

```go
tp := tmpl.New(os.DirFS("templates")).
    Autoload("components").
    MustParse()
```

## Load templates

### Load individual templates.

The template is named after the last file and the other files will be associated templates.

```go
tp := tmpl.New(os.DirFS("templates")).
    Load("partials/header", "partials/footer", "pages/index").
    MustParse()

// in the above example, the loaded template is named "pages/index"
// however "partials/header" and "partials/footer" are associated templates available within "pages/index"
```

### Load directory (recommended).

Load all templates like in a file-based router. By default the layout filename is `layout`.

```go
tp := tmpl.New(os.DirFS("templates")).
    LoadTree("pages").
    MustParse()
```

## Render templates

Tmpl has a default sync renderer and stream renderer. Renderers are not concurrent safe and should not be used across separate goroutines.

### Render template with struct (recommended)

A template is any type that implements `tmpl.Template`. The Template method returns the template name and template data.

> The template name and the template data are tightly coupled.

```go
type Home struct {
    Title string
}

func (h Home) Template() (string, any) {
    return "pages/home", h
}

func main() {
    templates := tmpl.New(os.DirFS("templates")).LoadTree("pages").MustParse()

    err := templates.Render(os.Stdout, Home{"Homepage"})
}
```

### Render template with `tmpl.Tmpl`

`tmpl.Tmpl` wraps any value in an internal struct that implements `tmpl.Template`.

> The template name and the template data are loosely coupled.

```go
func main() {
    templates := tmpl.New(os.DirFS("templates")).LoadTree("pages").MustParse()

    err := templates.Render(os.Stdout, tmpl.Tmpl("pages/home", tmpl.Map{
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
    Layout
    Username string
}

func (h Home) Template() (string, any) {
    return tmpl.Tmpl("pages/home", h.Layout, h).Template()
}

func main() {
    templates := tmpl.New(os.DirFS("templates")).LoadTree("pages").MustParse()

    err := tr.Render(os.Stdout, Home{Layout{"Homepage"}, "Johndoe"})
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

type Index struct {
    Username string
}

func (i Index) Template() (string, any) {
    return tmpl.Tmpl("pages/index", Layout{"Homepage"}, i).Template()
}
```
> When more than one data argument is provided `tmpl.Tmpl` composes the arguments as nested template data.

`templates/pages/index.html` is the entry point. It  defines the `content` block and renders the layout, passing all the template data to the layout.
`templates/pages/layout.html` receives the template data which should contain `.Data` and `.Child` fields (`.Data` is the layout data and `.Child` is the data for the child template) and renders the `content` block passing the child template data.

## Associated Templates

Associated templates are named templates within a loaded template. This is useful for rendering partials defined within the template. Take a look at the example below:

```html
<!-- templates/pages/index.html -->
<main>
    <p>Button example</p>
    {{ template "button" .Message }}
</main>

{{ define "button" }}
<button>{{ . }}</button>
{{ end }}
```

```go
// templates/pages/index.go

type Index struct { 
    Message string
}

func (i Index) Template() (string, any) {
    return "pages/index", i
}

type IndexButton struct {
    Text string
}

func (b IndexButton) AssociatedTemplate() (string, string, any) {
    return "pages/index", "button", b.Text
}
```

```go
// main.go

func main() {
    templates := tmpl.New(os.DirFS("templates")).LoadTree("pages").MustParse()

    templates.Render(os.Stdout, Index{"Click me"})
    // outputs:
    // <main>
    //      <p>Button example</p>
    //      <button>Click me</button>
    // </main>
    // <button>{{ . }}</button>

    templates.RenderAssociated(os.Stdout, IndexButton{"Click me"})
    // outputs:
    // <button>Click me</button>

    // Or using tmpl.AssociatedTmpl helper
    templates.RenderAssociated(os.Stdout, tmpl.AssociatedTmpl("pages/index", "button", "Press me"))
    // outputs:
    // <button>Press me</button>
}
```

## Clone templates

Clone templates to share similar configurations between templates.

```go
tp1 := tmpl.New(os.DirFS("templates")).SetExt("tmpl")
// tmpl extension applies to tp1, tp2 and tp3

tp2, err := tp1.Clone()
tp2.Autoload("components/ui")
// components/ui autoload applies only to tp2

// MustClone panics on clone error
tp3 := tp1.MustClone().Autoload("components/icons")
// components/icons autoload applies only to tp3
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

### clsx
Composes HTML class from successive arguments.
```html
{{ $class := clsx "flex gap-1 border p-3 rounded-lg"
    (eq .variant "error") "bg-red-100 text-red-500"
    (eq .variant "success") "bg-teal-100 text-teal-600"
    .class
}}
<div class="{{ $class }}">...</div>
```

### stream
Streams in templates that depend on an async value.
Streamed templates may optionally define pending and error templates as seen below.
See more about [HTML Streaming](html-streaming.md).
```html
<div>
    <h1>{{ stream "lazy" .LazyData }}</h1>
</div>

{{ define "lazy" }}
<p>Resolved: {{ . }}</p>
{{ end }}

<!-- pending template is optional -->
{{ define "lazy:pending" }}
<p>Loading...</p>
{{ end }}

<!-- error template is optional -->
{{ define "lazy:error" }}
<p>Failed: {{ . }}</p>
{{ end }}
```
