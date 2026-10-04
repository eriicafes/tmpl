package vite

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"maps"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Dev indicates Vite is running in development mode.
	Dev bool

	// DevOrigin is the origin of the Vite dev server.
	// It defaults to "http://localhost:5173".
	DevOrigin string

	// Output is the directory where vite build output will be placed.
	// It should match build.outDir and defaults to os.DirFS("dist").
	Output fs.FS

	// Base is the path the Vite application is served from and must start with a slash.
	// It should match base in vite.config and defaults to "/".
	Base string
}

type Vite struct {
	Config
	Manifest  Manifest
	devOrigin *url.URL
	importMap string
}

type Manifest map[string]ManifestChunk

type ManifestChunk struct {
	Src            string
	File           string
	Css            []string
	Assets         []string
	IsEntry        bool
	Name           string
	IsDynamicEntry bool
	Imports        []string
	DynamicImports []string
}

// New creates a new Vite instance.
// A non-nil error is returned if the Vite manifest is missing or malformed, or
// if a present import map is malformed, in production.
//
// To enable vite for your application render the vite tags in your template html head.
//
// {{ vite "path/to/input.js" }}
//
// Additionally for react using @vitejs/plugin-react render the following tag before the vite tags.
//
// {{ vite_react_refresh }}
//
// Vite entry points can be configured with the top-level input option in vite.config.
// Multiple entry points can be specified as follows:
//
// {{ vite "path/to/input1.js" "path/to/input2.js" }}
//
// You must enable the Vite manifest by setting build.manifest to true in vite.config.
// Run `vite build` and set Dev to false for production.
func New(config Config) (*Vite, error) {
	if config.DevOrigin == "" {
		config.DevOrigin = "http://localhost:5173"
	}
	if config.Output == nil {
		config.Output = os.DirFS("dist")
	}
	if !strings.HasPrefix(config.Base, "/") {
		config.Base = "/" + config.Base
	}
	v := &Vite{Config: config}
	if config.Dev {
		origin, err := url.Parse(config.DevOrigin)
		if err != nil || origin.Scheme == "" || origin.Host == "" {
			if err == nil {
				err = fmt.Errorf("must include a scheme and host")
			}
			return nil, fmt.Errorf("invalid Vite dev origin %q: %w", config.DevOrigin, err)
		}
		v.devOrigin = origin
		return v, nil
	}

	b, err := fs.ReadFile(config.Output, ".vite/manifest.json")
	if err != nil {
		return v, err
	}
	if err := json.Unmarshal(b, &v.Manifest); err != nil {
		return v, err
	}
	importMap, err := validateImportMap(config.Output)
	if err != nil {
		return v, err
	}
	v.importMap = importMap
	return v, nil
}

func validateImportMap(fsys fs.FS) (string, error) {
	b, err := fs.ReadFile(fsys, "importmap.json")
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var importMap any
	if err := json.Unmarshal(b, &importMap); err != nil {
		return "", fmt.Errorf("invalid Vite import map: %w", err)
	}
	b, err = json.Marshal(importMap)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Funcs returns vite helper functions for templates.
//
// vite returns required vite tags to be rendered in the html head.
// Usage: {{ vite "input" }} or {{ vite "input1" "input2" }} for multiple entry points.
//
// vite_public returns the absolute path for an asset in the public directory.
// Usage: {{ vite_public "logo.png" }}.
func (v *Vite) Funcs() template.FuncMap {
	return template.FuncMap{
		"vite":               v.ViteTags,
		"vite_entry":         v.EntryTags,
		"vite_public":        v.PublicPath,
		"vite_react_refresh": v.ReactRefresh,
		"vite_dev":           func() bool { return v.Dev },
	}
}

func (v *Vite) devURL(name string) string {
	path := strings.TrimSuffix(v.Base, "/") + "/" + strings.TrimPrefix(name, "/")
	u := *v.devOrigin
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	u.RawPath, u.RawQuery, u.Fragment = "", "", ""
	return u.String()
}

// PublicPath returns the absolute path for an asset in the public directory.
func (v *Vite) PublicPath(path string) string {
	if v.Dev {
		return v.devURL(path)
	}
	return strings.TrimSuffix(v.Base, "/") + "/" + strings.TrimPrefix(path, "/")
}

// ReactRefresh returns script for react refresh with @vitejs/plugin-react.
// During production this returns an empty string.
func (v *Vite) ReactRefresh() template.HTML {
	if !v.Dev {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<script type="module">
  import RefreshRuntime from '%s'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>`, v.devURL("@react-refresh")))
}

// ViteTags returns required vite tags to be rendered in the html head.
//
// https://vite.dev/guide/backend-integration.html
func (v *Vite) ViteTags(inputs ...string) (template.HTML, error) {
	tags := new(strings.Builder)
	if v.importMap != "" {
		appendTag(tags, fmt.Sprintf(`<script type="importmap">%s</script>`, v.importMap))
	}
	if v.Dev {
		appendTag(tags, scriptTag(v.devURL("@vite/client")))
	}
	for _, input := range inputs {
		entry, err := v.EntryTags(input)
		if err != nil {
			return "", err
		}
		appendTag(tags, string(entry))
	}
	return template.HTML(tags.String()), nil
}

// EntryTags returns the required Vite tags for one entry point.
func (v *Vite) EntryTags(input string) (template.HTML, error) {
	if v.Dev {
		path := v.devURL(input)
		if strings.HasSuffix(strings.ToLower(path), ".css") {
			return template.HTML(cssTag(path)), nil
		}
		return template.HTML(scriptTag(path)), nil
	}
	chunk, ok := v.Manifest[input]
	if !ok || !chunk.IsEntry {
		return "", fmt.Errorf("entry point %q does not exist in vite manifest", input)
	}
	tags := new(strings.Builder)
	for _, css := range chunk.Css {
		appendTag(tags, cssTag(v.PublicPath(css)))
	}
	chunks := importedChunks(v.Manifest, &chunk)
	for _, ch := range chunks {
		for _, css := range ch.Css {
			appendTag(tags, cssTag(v.PublicPath(css)))
		}
	}
	path := v.PublicPath(chunk.File)
	if strings.HasSuffix(strings.ToLower(path), ".css") {
		appendTag(tags, cssTag(path))
	} else {
		appendTag(tags, scriptTag(path))
	}
	for _, ch := range chunks {
		if !strings.HasSuffix(strings.ToLower(ch.File), ".css") {
			appendTag(tags, fmt.Sprintf(`<link rel="modulepreload" href="%s">`, template.HTMLEscapeString(v.PublicPath(ch.File))))
		}
	}
	return template.HTML(tags.String()), nil
}

func scriptTag(path string) string {
	return fmt.Sprintf(`<script type="module" src="%s"></script>`, template.HTMLEscapeString(path))
}

func cssTag(path string) string {
	return fmt.Sprintf(`<link rel="stylesheet" href="%s">`, template.HTMLEscapeString(path))
}

func appendTag(tags *strings.Builder, s string) (int, error) {
	if tags.Len() > 0 {
		tags.WriteString("\n\t")
	}
	return tags.WriteString(s)
}

func importedChunks(manifest Manifest, chunk *ManifestChunk) []*ManifestChunk {
	seen := make(map[string]bool)

	var getImportedChunks func(*ManifestChunk) []*ManifestChunk
	getImportedChunks = func(chunk *ManifestChunk) []*ManifestChunk {
		var chunks []*ManifestChunk
		for _, name := range chunk.Imports {
			if chunk, ok := manifest[name]; ok && !seen[name] {
				seen[name] = true
				chunks = append(chunks, getImportedChunks(&chunk)...)
				chunks = append(chunks, &chunk)
			}
		}
		return chunks
	}
	return getImportedChunks(chunk)
}

// ServePublic proxies requests to static assets to the vite server in development
// or serves the output directory in production for GET and HEAD requests.
// ServePublic executes the fallback handler for other requests or if the static asset is not found.
func (v *Vite) ServePublic(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			fallback.ServeHTTP(w, r)
			return
		}

		var handler http.Handler
		if v.Dev {
			handler = devProxyHandler(v.devOrigin)
		} else {
			handler = http.FileServerFS(filteredFS{v.Output})
			if v.Base != "/" {
				handler = http.StripPrefix(v.Base, handler)
			}
		}
		fw := &fallbackResponseWriter{w, false}
		handler.ServeHTTP(fw, r)
		if fw.notFound {
			fallback.ServeHTTP(w, r)
		}
	})
}

type fallbackResponseWriter struct {
	http.ResponseWriter
	notFound bool
}

func (fw *fallbackResponseWriter) Write(b []byte) (int, error) {
	if fw.notFound {
		// ignore write
		return len(b), nil
	}
	return fw.ResponseWriter.Write(b)
}

func (fw *fallbackResponseWriter) WriteHeader(statusCode int) {
	if statusCode == http.StatusNotFound {
		fw.notFound = true
		// delete previous headers
		h := fw.Header()
		for k := range h {
			h.Del(k)
		}
		return
	}
	fw.ResponseWriter.WriteHeader(statusCode)
}

type filteredFS struct{ fs fs.FS }

func (ff filteredFS) Open(name string) (fs.File, error) {
	file, err := ff.fs.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		dfile, derr := ff.fs.Open(filepath.Join(name, "index.html"))
		if derr != nil {
			file.Close()
			return nil, derr
		}
		dfile.Close()
	}
	return file, err
}

func devProxyHandler(origin *url.URL) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dest := *origin
		dest.Path = strings.TrimSuffix(origin.Path, "/") + r.URL.Path
		dest.RawPath = ""
		dest.RawQuery = r.URL.RawQuery
		dest.ForceQuery = r.URL.ForceQuery
		req, err := http.NewRequestWithContext(r.Context(), r.Method, dest.String(), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Header = r.Header.Clone()
		req.Host = origin.Host

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		h := w.Header()
		maps.Copy(h, resp.Header)
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
