package vite

import (
	"bytes"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestViteTagsProduction(t *testing.T) {
	v, err := New(Config{
		Base: "/app",
		Output: fstest.MapFS{
			".vite/manifest.json": {Data: []byte(`{
				"src/main.ts": {
					"file": "assets/main.js",
					"isEntry": true,
					"css": ["assets/main.css"],
					"imports": ["_shared.js"]
				},
				"_shared.js": {
					"file": "assets/shared.js",
					"css": ["assets/shared.css"]
				},
				"src/theme.css": {
					"file": "assets/theme.css",
					"isEntry": true
				}
			}`)},
			"importmap.json": {Data: []byte(`{"imports":{"theme":"./theme.js"}}`)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := v.ViteTags("src/main.ts", "src/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`<script type="importmap">{"imports":{"theme":"./theme.js"}}</script>`,
		`<link rel="stylesheet" href="/app/assets/main.css">`,
		`<link rel="stylesheet" href="/app/assets/shared.css">`,
		`<script type="module" src="/app/assets/main.js"></script>`,
		`<link rel="modulepreload" href="/app/assets/shared.js">`,
		`<link rel="stylesheet" href="/app/assets/theme.css">`,
	}, "\n\t")
	if got := string(tags); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestViteTagsDevelopment(t *testing.T) {
	v, err := New(Config{
		Dev:       true,
		DevOrigin: "https://vite.example.test:8443",
		Base:      "/app",
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := v.ViteTags("src/main.ts", "src/theme.css")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`<script type="module" src="https://vite.example.test:8443/app/@vite/client"></script>`,
		`<script type="module" src="https://vite.example.test:8443/app/src/main.ts"></script>`,
		`<link rel="stylesheet" href="https://vite.example.test:8443/app/src/theme.css">`,
	}, "\n\t")
	if got := string(tags); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEntryTagsDevelopment(t *testing.T) {
	v, err := New(Config{
		Dev:       true,
		DevOrigin: "https://vite.example.test:8443",
		Base:      "/app",
	})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := v.EntryTags("  src/profile.ts  ")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(tags), `<script type="module" src="https://vite.example.test:8443/app/src/profile.ts"></script>`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestEntryTagsProduction(t *testing.T) {
	v, err := New(Config{Output: fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{
			"src/profile.ts": {
				"file": "assets/profile.js",
				"isEntry": true,
				"css": ["assets/profile.css"],
				"imports": ["_shared.js"]
			},
			"_shared.js": {
				"file": "assets/shared.js",
				"css": ["assets/shared.css"]
			}
		}`)},
		"importmap.json": {Data: []byte(`{"imports":{"profile":"./profile.js"}}`)},
	}})
	if err != nil {
		t.Fatal(err)
	}

	tags, err := v.EntryTags("src/profile.ts")
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Join([]string{
		`<link rel="stylesheet" href="/assets/profile.css">`,
		`<link rel="stylesheet" href="/assets/shared.css">`,
		`<script type="module" src="/assets/profile.js"></script>`,
		`<link rel="modulepreload" href="/assets/shared.js">`,
	}, "\n\t")
	if got := string(tags); got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFuncs(t *testing.T) {
	v, err := New(Config{Dev: true, DevOrigin: "https://vite.example.test"})
	if err != nil {
		t.Fatal(err)
	}
	tmpl, err := template.New("test").Funcs(v.Funcs()).Parse(`{{ vite_entry "src/profile.ts" }}{{ if vite_dev }} development{{ end }}`)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := tmpl.Execute(&output, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := output.String(), `<script type="module" src="https://vite.example.test/src/profile.ts"></script> development`; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNewRejectsInvalidDevOrigin(t *testing.T) {
	_, err := New(Config{Dev: true, DevOrigin: "vite.example.test"})
	if err == nil {
		t.Fatal("expected invalid development origin error")
	}
}

func TestNewDevelopmentSkipsBuildFiles(t *testing.T) {
	v, err := New(Config{Dev: true, Output: fstest.MapFS{
		"importmap.json": {Data: []byte(`{`)},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if v.Manifest != nil {
		t.Fatalf("got manifest %v, want nil", v.Manifest)
	}
}

func TestNewRejectsMalformedImportMap(t *testing.T) {
	_, err := New(Config{Output: fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{}`)},
		"importmap.json":      {Data: []byte(`{`)},
	}})
	if err == nil {
		t.Fatal("expected malformed import map error")
	}
}

func TestServePublicPreservesQuery(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, r.URL.RequestURI())
	}))
	defer upstream.Close()

	v, err := New(Config{Dev: true, DevOrigin: upstream.URL})
	if err != nil {
		t.Fatal(err)
	}
	handler := v.ServePublic(http.NotFoundHandler())
	request := httptest.NewRequest(http.MethodGet, "/src/main.ts?raw&t=123", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "/src/main.ts?raw&t=123"; got != want {
		t.Fatalf("got request URI %q, want %q", got, want)
	}
}

func TestServePublicFallsBackForPost(t *testing.T) {
	v, err := New(Config{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	handler := v.ServePublic(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/assets/app.js", nil))
	if response.Code != http.StatusTeapot {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusTeapot)
	}
}

func TestServePublicFallsBackWhenDevAssetIsMissing(t *testing.T) {
	upstream := httptest.NewServer(http.NotFoundHandler())
	defer upstream.Close()

	v, err := New(Config{Dev: true, DevOrigin: upstream.URL})
	if err != nil {
		t.Fatal(err)
	}
	handler := v.ServePublic(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/missing.js", nil))
	if response.Code != http.StatusTeapot {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusTeapot)
	}
}

func TestServePublicProductionFile(t *testing.T) {
	v, err := New(Config{Output: fstest.MapFS{
		".vite/manifest.json": {Data: []byte(`{}`)},
		"assets/app.js":       {Data: []byte("console.log('app')")},
	}})
	if err != nil {
		t.Fatal(err)
	}
	handler := v.ServePublic(http.NotFoundHandler())
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("got status %d, want %d", response.Code, http.StatusOK)
	}
	if got, want := response.Body.String(), "console.log('app')"; got != want {
		t.Fatalf("got body %q, want %q", got, want)
	}
}
