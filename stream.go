package tmpl

import (
	"fmt"
	"html/template"
	"io"
	"strings"
)

// streamData is the result of an Async.
type streamData struct {
	ok   bool
	data any
}

type streamTemplate struct {
	streamData
	name string
	cid  uint32
}

func stream(t *template.Template, session *session, name string, av asyncValue) (template.HTML, error) {
	if av == nil {
		return "", fmt.Errorf("Async is nil")
	}
	if session == nil {
		return "", fmt.Errorf("stream called outside a render session")
	}
	if data, stored := av.getStored(); stored {
		return renderSync(t, name, data)
	}
	if !session.streaming {
		session.flush()
		data, resolved := av.get(session.ctx)
		if !resolved {
			return "", session.ctx.Err()
		}
		return renderSync(t, name, data)
	}
	return renderStream(t, session, name, av)
}

func renderSync(t *template.Template, name string, data streamData) (template.HTML, error) {
	if !data.ok {
		name += ":error"
		if t.Lookup(name) == nil {
			return "", nil
		}
	}
	return executeTemplate(t, name, data.data)
}

func renderStream(t *template.Template, session *session, name string, av asyncValue) (template.HTML, error) {
	session.startPending()
	cid := session.nextCID()
	session.pending++
	go func() {
		data, resolved := av.get(session.ctx)
		if !resolved {
			return
		}
		select {
		case session.ch <- streamTemplate{streamData: data, name: name, cid: cid}:
		case <-session.ctx.Done():
		}
	}()

	if t.Lookup(name+":pending") == nil {
		return pendingHTML(cid, ""), nil
	}
	pending, err := executeTemplate(t, name+":pending", nil)
	if err != nil {
		return "", err
	}
	return pendingHTML(cid, string(pending)), nil
}

func (s *session) await(t *template.Template) error {
	if s.pending == 0 {
		return nil
	}
	if _, err := io.WriteString(s.w, string(swapOOOSScript())); err != nil {
		return err
	}
	s.flush()

	for s.pending > 0 {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case streamed := <-s.ch:
			s.pending--
			contents, err := renderSync(t, streamed.name, streamed.streamData)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(s.w, string(resolvedHTML(streamed.cid, string(contents)))); err != nil {
				return err
			}
			s.flush()
		}
	}
	return nil
}

func executeTemplate(t *template.Template, name string, data any) (template.HTML, error) {
	html := new(strings.Builder)
	if err := t.ExecuteTemplate(html, name, data); err != nil {
		return "", err
	}
	return template.HTML(html.String()), nil
}

func pendingHTML(cid uint32, contents string) template.HTML {
	return template.HTML(fmt.Sprintf(`<!--tmpl:start:%d-->%s<!--tmpl:end:%d-->`, cid, contents, cid))
}

func resolvedHTML(cid uint32, contents string) template.HTML {
	return template.HTML(fmt.Sprintf(`<template data-tmpl-cid="%d">%s</template>
<script>swapOOOS("%d")</script>`, cid, contents, cid))
}

func swapOOOSScript() template.HTML {
	return template.HTML(`<script>
    function tmplComment(value) {
        const walker = document.createTreeWalker(document, NodeFilter.SHOW_COMMENT)
        while (walker.nextNode()) {
            if (walker.currentNode.data === value) return walker.currentNode
        }
        return null
    }
    function swapOOOS(cid) {
        const start = tmplComment("tmpl:start:" + cid)
        const end = tmplComment("tmpl:end:" + cid)
        const source = document.querySelector('template[data-tmpl-cid="' + cid + '"]')
        if (!start || !end || !source) return
        const range = document.createRange()
        range.setStartAfter(start); range.setEndBefore(end)
        range.deleteContents(); range.insertNode(source.content.cloneNode(true))
        start.remove(); end.remove(); source.remove(); document.currentScript?.remove()
    }
</script>`)
}
