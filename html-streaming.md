## HTML Streaming

Tmpl supports HTML streaming by writing html response as they become available. There are two rendering strategies with Tmpl. Consider the example template below to see the differences.

```html
<!DOCTYPE html>
<html lang="en">
<head>
    <title>Lazy page</title>
</head>
<body>
    {{ stream "lazy" .LazyData }}
</body>
</html>

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

### Render (Blocking)

When the sync renderer encounters an async value it flushes the written html and blocks until the async value resolves.

```go
package main

import (
	"fmt"
	"os"
	"github.com/eriicafes/tmpl"
)

type Index struct {
	LazyData tmpl.Async[string, error]
}

func (i Index) Tmpl() tmpl.Template {
	return tmpl.Tmpl("pages/index", i)
}

func main() {
	templates := tmpl.New(os.DirFS("templates")).
		LoadTree("pages").
		MustParse()

	page := Index{
		LazyData: tmpl.Go(func(value tmpl.Async[string, error]) {
			value.Ok("success")
		}),
	}

	err := templates.Render(os.Stdout, page)
	if err != nil {
		fmt.Println(err)
	}
}
```


### Stream (Out of Order Streaming)

When the stream renderer encounters an async value it immediately returns a pending fallback template and waits for the async value in a separate goroutine and then streams in the resolved template when it becomes available all in the same http response.

Out of Order Streaming improves server-side performance by sending as much HTML as possible and streaming in the dynamic parts of the page as they become available.

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/eriicafes/tmpl"
)

type Index struct {
	LazyData tmpl.Async[string, error]
}

func (i Index) Tmpl() tmpl.Template {
	return tmpl.Tmpl("pages/index", i)
}

func main() {
	templates := tmpl.New(os.DirFS("templates")).
		LoadTree("pages").
		MustParse()

	page := Index{
		LazyData: tmpl.Go(func(value tmpl.Async[string, error]) {
			time.Sleep(time.Second * 3)
			value.Ok("success")
		}),
	}

	err := templates.Stream(context.Background(), os.Stdout, page)
	if err != nil {
		fmt.Println(err)
	}
}
```

Under the hood, Tmpl surrounds a pending async value with comment boundaries and waits for the value in a separate goroutine. When the value is available, it sends the resolved template to the response stream and a client-side script replaces the content between those boundaries.

The pending and resolved templates must be valid children of the element containing `stream`. For example, a stream inside an `h1` should resolve to text or phrasing content, not a `p`. Raw-text elements such as `script`, `style`, `title`, and `textarea` cannot contain streams.
