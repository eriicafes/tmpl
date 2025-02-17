# Go Tmpl

### Simple and powerful go templates with zero code generation.

Tmpl makes go templates composable, easy to work with and predictable, suitable for rendering pages with layouts or rendering page partials with frameworks like [htmx.org](https://htmx.org).

Tmpl organizes the way load [go templates](https://pkg.go.dev/html/template) and provides some essential template funcs and patterns.

Tmpl provides first-party support for [Vite](https://vite.dev).
See [Vite integration](vite/README.md) section.

## Installation

```sh
go get github.com/eriicafes/tmpl
```

## Features

- Render templates with structs
- Automatic templates loading
- Supports template [layouts](#layouts) and [slots](#tmpl--slot)
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
fs := os.DirFS("templates")
tp := tmpl.New(fs).
    SetExt("tmpl"). // default is "html"
    SetLayoutFilename("_layout"). // default is "layout"
    Funcs(funcMaps...). // register template funcs here
    OnLoad(func(name string, t *template.Template) {
        // called on template load, before template is parsed
        t.Funcs(template.FuncMap{})
    }).
    MustParse()
```

## Load templates

### Load individual templates.

The template is named after the last file and the other files will be associated templates.

```go
fs := os.DirFS("templates")
tp := tmpl.New(fs).
    Load("partials/header", "partials/footer", "pages/index").
    MustParse()

// in the above example, the loaded template is named "pages/index"
// however "partials/header" and "partials/footer" are associated templates available within "pages/index"
```

### Load directory (recommended).

Load all templates like in a file-based router. By default the layout filename for each path segment is `layout`.
Layouts templates are available as associated templates within the loaded template.

```go
fs := os.DirFS("templates")
tp := tmpl.New(fs).
    LoadTree("pages").
    MustParse()
```

## Autoload templates

Autoloaded templates are available as [associated templates](#render-associated-templates) in all templates.

```go
fs := os.DirFS("templates")
tp := tmpl.New(fs).
    Autoload("components").
    MustParse()
```

## Render templates

A template is any type that implements `tmpl.Template`.
Add a Tmpl method on your custom type and return a template definition.

> Tmpl has a default sync renderer and a stream renderer. [See streaming guide](html-streaming.md).

### Render template inline

```go
func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).LoadTree("pages").MustParse()

    err := tp.Render(os.Stdout, tmpl.Tmpl("pages/home", tmpl.Map{
        "Title": "Homepage",
    }))
}
```

### Render template with types (recommended)

Implement the `tmpl.Template` interface for your custom type and use it to render.

```go
type Home struct {
    Title string
}

func (h Home) Tmpl() tmpl.Template {
    return tmpl.Tmpl("pages/home", h)
}

func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).LoadTree("pages").MustParse()

    err := tp.Render(os.Stdout, Home{"Homepage"})
}
```

## Render associated templates

Associated templates are named templates within a template.
This is useful for rendering layouts, autoloaded templates or partials defined within the template.
Take a look at the example below:

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
// main.go

type Index struct { 
    Message string
}

func (i Index) Tmpl() tmpl.Template {
    return tmpl.Tmpl("pages/index", i)
}

type Button struct {
    Text string
}

func (b Button) Tmpl() tmpl.Template {
    return tmpl.Associated("pages/index", "button", b.Text)
}

func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).LoadTree("pages").MustParse()

    tp.Render(os.Stdout, Index{"Click me"})
    // outputs:
    // <main>
    //      <p>Button example</p>
    //      <button>Click me</button>
    // </main>

    tp.Render(os.Stdout, IndexButton{"Click me"})
    // outputs:
    // <button>Click me</button>

    // Or inline
    tp.Render(os.Stdout, tmpl.Associated("pages/index", "button", "Press me"))
    // outputs:
    // <button>Press me</button>
}
```

## Layouts

When using [LoadTree](#load-directory-recommended) layout templates for each path segment are available as associated templates.
Render layouts as regular associated templates and render the dynamic children using slot.
See the example below:

### Directory structure

```text
/
├── templates/
│   ├── components/
│   ├── pages/
│   │   │── index.html
│   │   └── layout.html
```

### HTML structure

```html
<!-- templates/pages/layout.html -->
<!DOCTYPE html>
<html lang="en">
  <head>
    <title>{{ .Title }}</title>
  </head>
  <body>
    <header>
      <h1>{{ .Title }}</h1>
    </header>
    {{ slot .Children }}
  </body>
</html>
```

```html
<!-- templates/pages/index.html -->
<main>
  <p>Hello {{ .Username }}</p>
</main>
```

### Render layouts inline

```go
// main.go

func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).LoadTree("pages").MustParse()

    err := tp.Render(os.Stdout, tmpl.Associated("pages/index", "pages/layout", tmpl.Map{
		"Title":    "Homepage",
		"Children": tmpl.Tmpl("pages/index", tmpl.Map{"Username": "Bob"}),
	}))
}
```

### Render layouts with struct (recommended)

When using structs you can embed `tmpl.Children` to the layout template struct and use `tmpl.Wrap` to compose layouts.

```go
// main.go

type Layout struct {
    tmpl.Children
    Title string
}

func (l Layout) Tmpl() tmpl.Template {
    return tmpl.Associated(l.Base(), "pages/layout", l)
}

type Index struct {
    Layout
    Username string
}

func (i Index) Tmpl() tmpl.Template {
    return tmpl.Wrap(&i.Layout, tmpl.Tmpl("pages/index", i))
}

func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).LoadTree("pages").MustParse()

    err := tp.Render(os.Stdout, Index{
        Layout: Layout{
            Title: "Homepage",
        },
        Username: "Bob",
    })
}
```

## Single File Templates
If you always need [typed templates](#render-template-with-types-recommended) you might want to colocate template types and content in a single go file.

Set template extension to go files and define the template content using `tmpl.Define`.
`tmpl.Define` must use backticks and must be executed exactly once in the init function of the go file.

> When a loading a template file that has a .go extension tmpl will only extract the arguments of a `tmpl.Define` function call.

```go
// templates/pages/home.go

package templates

import "github.com/eriicafes/tmpl"

type Home struct {
	Title string
}

func (h Home) Tmpl() tmpl.Template {
	return tmpl.Tmpl("pages/home", h)
}

func init() {
	tmpl.Define(`
	<!DOCTYPE html>
	<html lang="en">
	<head>
		<title>{{ .Title }}</title>
	</head>
	<body>
		<h1>{{ .Title }}</h1>
	</body>
	</html>
	`)
}
```

```go
// main.go

func main() {
    fs := os.DirFS("templates")
    tp := tmpl.New(fs).SetExt("go").LoadTree("pages").MustParse()

    err := tp.Render(os.Stdout, Home{"Homepage"})
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

### tmpl & slot
Go Templates does not have a clear way of using slots so you have to rely on
overriding associated template definitions which has several pitfalls.

Use `tmpl` to create a `tmpl.Template` inside templates.

`tmpl [template name] [template data]`

Use `slot` to execute slotted content. Slotted content can be a `tmpl.Template` or string.

`slot [slotted content]`

```html
<!-- button.html -->
<button class="{{ .class }}">
    {{ slot .children }}
</button>

<!-- select.html -->
<select name="{{ .name }}">
    {{ slot .children }}
</select>

<!-- index.html -->
<html>
    <head>...</head>
    <body>
        <form>
            <input name="message" type="text" placeholder="Your message" />

            {{ template "select" map
                "name" "subject"
                "children" (tmpl "subject-options" .Options) // template slot
            }}
            {{ define "subject-options" }}
                {{ range . }}
                <option>{{ . }}</option>
                {{ end }}
            {{ end }}

            {{ template "button" map
                "class" "px-4 py-2 rounded-md bg-black text-white"
                "children" "Submit form" // string slot
            }}
        </form>
    </body>
</html>
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

## Clone templates

Clone templates to share similar configurations between templates.

```go
fs := os.DirFS("templates")
tp1 := tmpl.New(fs).SetExt("tmpl")
// tmpl extension applies to tp1, tp2 and tp3

tp2, err := tp1.Clone()
tp2.Autoload("components/ui")
// components/ui autoload applies only to tp2

// MustClone panics on clone error
tp3 := tp1.MustClone().Autoload("components/icons")
// components/icons autoload applies only to tp3
```
