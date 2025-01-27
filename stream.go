package tmpl

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"sync"
)

// streamData is the shape used to resolve an async value,
// it contains the resolved data and a bool flag to indicate success or error.
type streamData struct {
	ok   bool
	data any
}

// streamTp is the shape sent to the stream channel which contains the name of the template to render and it's data.
type streamTp struct {
	streamData
	name string
	cid  int32
}

// streamController controls streams from templates resolved with an AsyncValue.
type streamController struct {
	ch  chan streamTp
	wg  *sync.WaitGroup
	cid int32
}

func newStreamController() *streamController {
	return &streamController{make(chan streamTp), new(sync.WaitGroup), 0}
}

func (c *streamController) nextCID() int32 {
	c.cid++
	return c.cid
}

func stream(t *template.Template, name string, av asyncValuer) (template.HTML, error) {
	if av == nil {
		return "", fmt.Errorf("AsyncValue is nil")
	}
	r := getRenderer(av.renderer())
	if r == nil {
		return "", fmt.Errorf("AsyncValue must be initialized with non-nil renderer")
	}
	// render template with cached data if available
	if data, cached := av.getCached(); cached {
		return renderSync(t, name, data)
	}
	if r.stream == nil {
		// flush available html
		if f, ok := r.w.(http.Flusher); ok {
			f.Flush()
		}
		// block until channel data is available before rendering template.
		return renderSync(t, name, av.get())
	}
	return renderStream(t, r.stream, name, av)
}

func renderSync(t *template.Template, name string, d streamData) (template.HTML, error) {
	html := new(strings.Builder)
	// if not ok render error template instead
	if !d.ok {
		name += ":error"
	}
	err := t.ExecuteTemplate(html, name, d.data)
	if err != nil && !d.ok {
		// silence execute error when rendering error template
		return "", nil
	}
	return template.HTML(html.String()), err
}

// renderStream immediately renders a pending template if channel data is not yet available,
// and streams in resolved content as they become avalable.
func renderStream(t *template.Template, stream *streamController, name string, av asyncValuer) (template.HTML, error) {
	select {
	case <-av.doneChan():
		data, _ := av.getCached()
		return renderSync(t, name, data)
	default:
		cid := stream.nextCID()
		stream.wg.Add(1)
		// queue render by sending template data to channel when available
		go func() {
			stream.ch <- streamTp{av.get(), name, cid}
		}()
		// immediately render pending template or empty slot if no pending template
		html := new(strings.Builder)
		if err := t.ExecuteTemplate(html, name+":pending", nil); err != nil {
			return pendingHTML(cid, ""), nil
		}
		return pendingHTML(cid, html.String()), nil
	}
}

func awaitStream(w io.Writer, t *template.Template, stream *streamController) error {
	// append swap script
	w.Write([]byte(swapOOOSScript()))

	// flush available html
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	doneCh := make(chan struct{})
	// waiting goroutine
	go func() {
		stream.wg.Wait()
		close(doneCh)
	}()

	var rerr error
	for {
		select {
		case streamTp := <-stream.ch:
			if rerr != nil {
				// return early to drain channel and free waiting goroutine
				stream.wg.Done()
				continue
			}
			// if not ok render error template instead
			if !streamTp.ok {
				streamTp.name += ":error"
			}
			// render template for each template data received on channel
			html := new(strings.Builder)
			err := t.ExecuteTemplate(html, streamTp.name, streamTp.data)
			if err != nil && !streamTp.ok {
				// silence execute error when rendering error template
				_, err = w.Write([]byte(resolvedHTML(streamTp.cid, "")))
			} else if err == nil {
				_, err = w.Write([]byte(resolvedHTML(streamTp.cid, html.String())))
			}
			// flush resolved html
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			if err != nil {
				rerr = err
			}
			stream.wg.Done()
		case <-doneCh:
			return rerr
		}
	}
}

func pendingHTML(cid int32, contents string) template.HTML {
	return template.HTML(fmt.Sprintf(`<div data-tmpl-cid="%d">
	%s
</div>`, cid, contents))
}

func resolvedHTML(cid int32, contents string) template.HTML {
	return template.HTML(fmt.Sprintf(`<template data-tmpl-cid="%d">
	%s
</template>
<script>swapOOOS("%d")</script>`, cid, contents, cid))
}

func swapOOOSScript() template.HTML {
	return template.HTML(fmt.Sprintf(`<script>
    function swapOOOS(cid) {
        const target = document.querySelector(%s), template = document.querySelector(%s), clone = template.content.cloneNode(true)
        target.replaceWith(clone); template.remove(); document.currentScript.remove();
    }
</script>`, "`[data-tmpl-cid=\"${cid}\"]`", "`template[data-tmpl-cid=\"${cid}\"]`"))
}
