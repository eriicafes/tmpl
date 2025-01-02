package tmpl

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

// streamTp is the shape sent to the stream channel to decide which template to render and it's data.
type streamTp struct {
	streamData
	name string
	cid  int32
}

// streamData is the shape used to resolve an async value,
// it contains the resolved data and a bool flag to indicate success or error.
type streamData struct {
	ok   bool
	data any
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

func stream(name string, av asyncValueRenderer) (template.HTML, error) {
	if av == nil {
		return "", fmt.Errorf("AsyncValue is nil")
	}
	r := av.renderer()
	if r == nil {
		return "", fmt.Errorf("AsyncValue must be initialized with non-nil renderer")
	}
	// render template with cached data if available
	if data, cached := av.getCached(); cached {
		return r.renderSync(name, data)
	}
	if r.stream == nil {
		// flush available html
		if f, ok := r.w.(http.Flusher); ok {
			f.Flush()
		}
		// block until channel data is available before rendering template.
		return r.renderSync(name, av.get())
	}
	return r.renderStream(name, av)
}

func (r *renderer) renderSync(name string, d streamData) (template.HTML, error) {
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
func (r *renderer) renderStream(name string, av asyncValueRenderer) (template.HTML, error) {
	select {
	case <-av.readyChan():
		data, _ := av.getCached()
		return r.renderSync(name, data)
	default:
		cid := r.stream.nextCID()
		r.stream.wg.Add(1)
		// queue render by sending template data to channel when available
		go func() {
			r.stream.ch <- streamTp{av.get(), name, cid}
		}()
		// immediately render pending template or empty slot if no pending template
		html := new(strings.Builder)
		if err := r.tp.ExecuteTemplate(html, name+":pending", nil); err != nil {
			return pendingHTML(cid, ""), nil
		}
		return pendingHTML(cid, html.String()), nil
	}
}

func (r *renderer) awaitStream() error {
	// append swap script
	r.w.Write([]byte(swapOOOSScript()))

	// flush available html
	if f, ok := r.w.(http.Flusher); ok {
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
				_, err = r.w.Write([]byte(resolvedHTML(streamTp.cid, "")))
			} else if err == nil {
				_, err = r.w.Write([]byte(resolvedHTML(streamTp.cid, html.String())))
			}
			// flush resolved html
			if f, ok := r.w.(http.Flusher); ok {
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
