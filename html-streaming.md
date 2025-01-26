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

### Sync Renderer (Blocking)

When the sync renderer encounters an async value it flushes the written html and blocks until the async value resolves.

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/eriicafes/tmpl"
)

type Index struct {
	LazyData tmpl.AsyncValue[string, error]
}

func (i Index) Template() (string, any) {
	return "pages/index", i
}

func main() {
	templates := tmpl.New(os.DirFS("templates")).
		LoadTree("pages").
		MustParse()

	tr := templates.SyncRenderer() // using the sync renderer
	page := Index{
		LazyData: tmpl.NewAsyncValue[string, error](tr),
	}

	go func() {
		time.Sleep(time.Second * 3)
		page.LazyData.Ok("success")
	}()

	err := tr.Render(os.Stdout, page)
	if err != nil {
		fmt.Println(err)
	}
}
```


### Stream Renderer (Out of Order Streaming)

When the stream renderer encounters an async value it immediately returns a pending fallback template and waits for the async value in a separate goroutine and then streams in the resolved template when it becomes available all in the same http response.

Out of Order Streaming improves server-side performance by sending as much HTML as possible and streaming in the dynamic parts of the page as they become available.

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/eriicafes/tmpl"
)

type Index struct {
	LazyData tmpl.AsyncValue[string, error]
}

func (i Index) Template() (string, any) {
	return "pages/index", i
}

func main() {
	templates := tmpl.New(os.DirFS("templates")).
		LoadTree("pages").
		MustParse()

	tr := templates.StreamRenderer() // using the stream renderer
	page := Index{
		LazyData: tmpl.NewAsyncValue[string, error](tr),
	}

	go func() {
		time.Sleep(time.Second * 3)
		page.LazyData.Ok("success")
	}()

	err := tr.Render(os.Stdout, page)
	if err != nil {
		fmt.Println(err)
	}
}
```

Under the hood when you stream a template with a pending async value, Tmpl renders the pending template with a div which has a data-tmpl-cid attribute and then waits for the async value in a separate goroutine. When the async value is available it executes the template and sends it to the html response stream after which a client side script swaps the pending template with the resolved template.