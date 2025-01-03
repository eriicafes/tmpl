package vite

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	// Dev indicates vite is running in developement mode, defaults to false.
	Dev bool

	// Port is the port the vite dev server is running on.
	// Should match viteConfig.server.port, defaults to "5173".
	Port string

	// Output is the directory where vite build output will be placed.
	// Should match viteConfig.build.outDir, defaults to os.DirFS("dist").
	Output fs.FS

	// Base is the path the vite applitcation is served from, it should start with a slash.
	// Should match viteConfig.base, defaults to "/".
	Base string
}

type Vite struct {
	Config
	Manifest Manifest
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

// New creates a new vite instance.
// A a non-nil error is returned if the vite manifest is missing or malformed when running in production.
//
// To enable vite for your application render the vite tags in your template html head.
//
// {{ vite "path/to/input.js" }}
//
// Additionally for react using @vitejs/plugin-react render the following tag before the vite tags.
//
// {{ vite_react_refresh }}
//
// Vite entry point can be configured using viteConfig.build.rollupOptions.input in your vite config.
// Multiple entry points can be specified as follows:
//
// {{ vite "path/to/input1.js" "path/to/input2.js" }}
//
// You must enable vite manifest by setting viteConfig.build.manifest to true in your vite config.
// Run `vite build` and set Dev to false for production.
func New(config Config) (*Vite, error) {
	if config.Port == "" {
		config.Port = "5173"
	}
	if config.Output == nil {
		config.Output = os.DirFS("dist")
	}
	if !strings.HasPrefix(config.Base, "/") {
		config.Base = "/" + config.Base
	}
	m := make(Manifest)
	var err error
	if !config.Dev {
		b, rerr := fs.ReadFile(config.Output, ".vite/manifest.json")
		err = rerr
		if err == nil {
			err = json.Unmarshal(b, &m)
		}
	}
	return &Vite{
		Manifest: m,
		Config:   config,
	}, err
}

// Funcs returns vite helper functions for templates.
//
// vite returns required vite tags to be rendered in the html head.
// Usage: {{ vite "input" }} or {{ vite "input1" "input2" }} for multiple entry points.
//
// public returns the absolute path for an asset in the public directory.
// Usage: {{ public "logo.png" }}.
//
// assets returns the absolute path for an asset in vite entry point viteConfig.build.rollupOptions.input.
// Use for assets that are not already required when rendering vite tags.
// Usage: {{ assets "src/main.ts" }}.
func (v *Vite) Funcs() template.FuncMap {
	return template.FuncMap{
		"vite":               v.ViteTags,
		"vite_public":        v.PublicPath,
		"vite_asset":         v.AssetPath,
		"vite_react_refresh": v.reactRefresh,
	}
}

func (v *Vite) devUrl(path string) string {
	base := fmt.Sprintf("http://localhost:%s", v.Port) + v.Base
	return strings.TrimSuffix(base, "/") + "/" + strings.TrimPrefix(path, "/")
}

// PublicPath returns the absolute path for an asset in the public directory.
func (v *Vite) PublicPath(path string) string {
	return strings.TrimSuffix(v.Base, "/") + "/" + strings.TrimPrefix(path, "/")
}

// AssetPath returns the absolute path for an asset in the vite entry point viteConfig.build.rollupOptions.input.
// During development AssetPath returns the file name as is.
func (v *Vite) AssetPath(name string) (string, error) {
	if v.Dev {
		return name, nil
	}
	chunk, ok := v.Manifest[name]
	if !ok {
		return "", fmt.Errorf("asset %q does not exist in vite manifest", name)
	}
	return strings.TrimSuffix(v.Base, "/") + "/" + chunk.File, nil
}

// ViteTags returns required vite tags to be rendered in the html head.
//
// https://vite.dev/guide/backend-integration.html
func (v *Vite) ViteTags(inputs ...string) (template.HTML, error) {
	tags := new(strings.Builder)
	if v.Dev {
		appendTag(tags, fmt.Sprintf(`<script type="module" src="%s"></script>`, v.devUrl("@vite/client")))
		for _, input := range inputs {
			path, err := v.AssetPath(input)
			if err != nil {
				return "", err
			}
			appendTag(tags, fmt.Sprintf(`<script type="module" src="%s"></script>`, v.devUrl(path)))
		}
		return template.HTML(tags.String()), nil
	}
	for _, input := range inputs {
		chunk, ok := v.Manifest[input]
		if !ok || !chunk.IsEntry {
			return "", fmt.Errorf("entry point %q does not exist in vite manifest", input)
		}
		for _, css := range chunk.Css {
			appendTag(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, v.PublicPath(css)))
		}
		chunks := importedChunks(v.Manifest, &chunk)
		for _, ch := range chunks {
			for _, css := range ch.Css {
				appendTag(tags, fmt.Sprintf(`<link rel="stylesheet" href="%s" />`, v.PublicPath(css)))
			}
		}
		appendTag(tags, fmt.Sprintf(`<script type="module" src="%s"></script>`, v.PublicPath(chunk.File)))
		for _, ch := range chunks {
			appendTag(tags, fmt.Sprintf(`<link rel="modulepreload" href="%s" />`, v.PublicPath(ch.File)))
		}
	}
	return template.HTML(tags.String()), nil
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

// reactRefresh returns script for react refresh with @vitejs/plugin-react.
// During production this returns an empty string.
func (v *Vite) reactRefresh() template.HTML {
	if !v.Dev {
		return ""
	}
	return template.HTML(fmt.Sprintf(`<script type="module">
  import RefreshRuntime from '%s'
  RefreshRuntime.injectIntoGlobalHook(window)
  window.$RefreshReg$ = () => {}
  window.$RefreshSig$ = () => (type) => type
  window.__vite_plugin_react_preamble_installed__ = true
</script>`, v.devUrl("@react-refresh")))
}

// ServePublic proxies requests to static assets to the vite server in development
// or serves the output directory in production for GET requests.
// ServePublic executes the fallback handler for non GET requests or if the static asset is not found.
func (v *Vite) ServePublic(fallback http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			fallback.ServeHTTP(w, r)
			return
		}

		var handler http.Handler
		if v.Dev {
			handler = devProxyHandler(fmt.Sprintf("localhost:%s", v.Port))
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

func devProxyHandler(host string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dest := "http://" + host + r.URL.Path
		req, err := http.NewRequestWithContext(r.Context(), r.Method, dest, r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		req.Header = r.Header.Clone()
		req.Host = r.Host

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer resp.Body.Close()

		h := w.Header()
		for k, v := range resp.Header {
			h[k] = v
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	})
}
