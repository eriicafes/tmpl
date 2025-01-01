package tmpl

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strings"
	"sync"
)

// streamTp is the shape passed to the stream channel to decide which template to render and it's data.
type streamTp struct {
	streamData
	name string
	cid  int32
}

// streamData is the shape that is used to resolve an async value,
// it contains the resolved data and a bool flag to indicate success or error.
type streamData struct {
	ok   bool
	data any
}

// streamCache stores resolved values from an AsyncValue.
// It caches the channel data and closes the channel.
// Further calls to get from the same channel will return the cached value.
type streamCache map[<-chan streamData]streamData

func (c streamCache) get(streamData chan streamData) streamData {
	data, ok := <-streamData
	if !ok {
		return c[streamData]
	}
	c[streamData] = data
	close(streamData)
	return data
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

func stream(name string, v asyncValueRenderer) (template.HTML, error) {
	if v == nil {
		return "", fmt.Errorf("AsyncValue is nil")
	}
	r, ch := v.renderer()
	if r == nil {
		return "", fmt.Errorf("AsyncValue must be initialized with non-nil renderer")
	}
	rr := r.Unwrap()
	if rr.stream == nil {
		// blocks until channel data is available before rendering template.
		// data from the same channel reference is cached.
		return renderSync(rr, name, rr.cache.get(ch))
	}
	return renderStream(rr, name, ch)
}

func renderSync(r *renderer, name string, d streamData) (template.HTML, error) {
	html := new(strings.Builder)
	// if not ok render error template instead
	if !d.ok {
		name += ":error"
	}
	err := r.tp.ExecuteTemplate(html, name, d.data)
	if err != nil && !d.ok {
		// silence execute error when rendering error template
		return "", nil
	}
	return template.HTML(html.String()), err
}

// renderStream immediately renders a pending template if channel data is not yet available,
// and streams in resolved content as they become avalable.
func renderStream(r *renderer, name string, ch chan streamData) (template.HTML, error) {
	cached, ok := r.cache[ch]
	if ok {
		// render template with cached data
		return renderSync(r, name, cached)
	}
	select {
	case d := <-ch:
		// render template with ready channel data
		return renderSync(r, name, d)
	default:
		cid := r.stream.nextCID()
		r.stream.wg.Add(1)
		// queue render by sending template data to channel when available
		go func() {
			r.stream.ch <- streamTp{r.cache.get(ch), name, cid}
		}()
		// immediately render pending template or empty slot if no pending template
		html := new(strings.Builder)
		if err := r.tp.ExecuteTemplate(html, name+":pending", nil); err != nil {
			return pendingHTML(cid, ""), nil
		}
		return pendingHTML(cid, html.String()), nil
	}
}

func awaitStream(r *renderer, w io.Writer) error {
	// append swap script
	w.Write([]byte(swapOOOSScript()))

	// flush available html
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	doneCh := make(chan struct{})
	// waiting goroutine
	go func() {
		r.stream.wg.Wait()
		close(doneCh)
	}()

	var rerr error
	for {
		select {
		case streamTp := <-r.stream.ch:
			if rerr != nil {
				// return early to drain channel and free waiting goroutine
				r.stream.wg.Done()
				continue
			}
			// if not ok render error template instead
			if !streamTp.ok {
				streamTp.name += ":error"
			}
			// render template for each template data received on channel
			html := new(strings.Builder)
			err := r.tp.ExecuteTemplate(html, streamTp.name, streamTp.data)
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
			r.stream.wg.Done()
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
