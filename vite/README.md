## Vite Integration

Vite is a powerful frontend build tool which provides a handful of tools including bundling of static assets, CSS, JS and TypeScript source code for development and production.

Tmpl provides first-party support for [Vite](https://vite.dev). In development Tmpl proxies requests to static assets to the vite development server while in production it serves the vite output directory.

Configure vite in 3 easy steps:

#### 1. Create vite instance, add templates funcs and setup middleware.

```go
// main.go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/eriicafes/tmpl"
	"github.com/eriicafes/tmpl/vite"
)

func main() {
	v, err := vite.New(vite.Config{Dev: true}) // <-- create Vite instance
	if err != nil {
		panic(err)
	}
	templates := tmpl.New(os.DirFS("templates")).
		Funcs(v.Funcs()). // <-- register vite template funcs
		LoadTree("pages").
		MustParse()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := templates.Render(w, tmpl.Tmpl("pages/index", nil))
		if err != nil {
			fmt.Println(err)
		}
	})
	http.Handle("/", v.ServePublic(handler)) // <-- wrap handler with vite middleware
	http.ListenAndServe(":8000", nil)
}
```

#### 2. Update vite config.

Enable the Vite manifest and configure the entry point. Vite 8 recommends the
top-level `input` option so development and production use the same entry.

```ts
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  input: "/src/main.tsx",
  plugins: [react()],
  server: {
    cors: {
      origin: "http://localhost:8000",
    },
  },
  build: {
    manifest: true, // <-- enable vite manifest
  },
})
```

This integration uses a custom JavaScript entry rather than an HTML entry. Add
the module-preload polyfill at the beginning of that entry unless you disabled
Vite's module-preload polyfill:

```ts
import "vite/modulepreload-polyfill"
```

When the Go application is not at the default local origin, set `DevOrigin` to
the complete Vite server origin and allow the Go application's browser origin
with `server.cors`:

```go
vite.New(vite.Config{
    Dev:       true,
    DevOrigin: "https://vite.example.test:5173",
})
```

#### 3. Return html document with the vite tags.

Render the vite tags in your template html head.

`{{ vite "path/to/input.js" "path/to/input.css" }}`

Additionally for React using `@vitejs/plugin-react` render the react refresh script before the vite tags.

`{{ vite_react_refresh }}`

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + React + TS</title>
    {{ vite_react_refresh }} <!-- for React only -->
    {{ vite "src/main.tsx" }}
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```

### Vite Funcs

#### vite

vite returns the required Vite tags. For each input it returns a module script
or stylesheet tag matching the input's output type.

```html
<!doctype html>
<html lang="en">
  <head>
    <title>Vite + TS</title>
    <!-- executing this -->
    {{ vite "src/main.ts" }}

    <!-- returns this in development -->
    <script type="module" src="http://localhost:5173/@vite/client"></script>
    <script type="module" src="http://localhost:5173/src/main.ts"></script>

    <!-- returns this in production -->
    <script type="module" src="/assets/main.js"></script>
  </head>
</html>
```

#### vite_public

vite_public references static assets relative to the vite publicDir.
In development requests are proxied to the vite development server.
In production the ServePublic middleware serves the vite output directory.

```html
<!doctype html>
<html lang="en">
  <head>...</head>
  <body>
    <!-- executing this -->
    <img src="{{ vite_public "images/logo.png" }}" width="200" height="200" />

    <!-- returns this in development -->
    <img src="/images/logo.png" width="200" height="200" />

    <!-- returns this in production -->
    <img src="/images/logo.png" width="200" height="200" />
  </body>
</html>
```

#### vite_entry

vite_entry returns the tags for one Vite entry without repeating the development
client or import map emitted by vite. Use it for a script or stylesheet that
only a particular page needs.

```html
<!-- layout -->
{{ vite "app/main.ts" "app/main.css" }}

<!-- profile page -->
{{ vite_entry "app/pages/profile.ts" }}
```

#### vite_react_refresh

vite_react_refresh returns the react refresh preamble.
If you are using React with `@vitejs/plugin-react`, you'll need to add this before the vite tags.

```html
<!doctype html>
<html lang="en">
  <head>
    <!-- executing this -->
    {{ vite_react_refresh }}

    <!-- returns this in development -->
    <script type="module">
      import RefreshRuntime from 'http://localhost:5173/@react-refresh'
      RefreshRuntime.injectIntoGlobalHook(window)
      window.$RefreshReg$ = () => {}
      window.$RefreshSig$ = () => (type) => type
      window.__vite_plugin_react_preamble_installed__ = true
    </script>

    <!-- returns nothing in production -->
  </head>
</html>
```

#### vite_dev

vite_dev reports whether Vite is running in development mode. Use it only for
development-only template behavior.

```html
{{ if vite_dev }}
  <script>console.info("development mode")</script>
{{ end }}
```

### Vite 8 import maps

When Vite's experimental `build.chunkImportMap` option is enabled, it writes
`importmap.json` beside the build output. This integration loads that file and
emits its import map before Vite's module and preload tags.

### Deploying under nested path

If you are deploying the vite application under a nested path make sure to specify the base option in both the vite config and in Go.

Specify base in Go.
```go
// main.go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/eriicafes/tmpl"
	"github.com/eriicafes/tmpl/vite"
)

func main() {
	v, err := vite.New(vite.Config{Dev: true, Base: "/app"}) // <-- specify base
	if err != nil {
		panic(err)
	}
	templates := tmpl.New(os.DirFS("templates")).
		Funcs(v.Funcs()).
		LoadTree("pages").
		MustParse()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := templates.Render(w, tmpl.Tmpl("pages/index", nil))
		if err != nil {
			fmt.Println(err)
		}
	})
	http.Handle("/app/", v.ServePublic(handler)) // <-- mount vite app under base
	http.ListenAndServe(":8000", nil)
}
```

Specify base in vite config.
```ts
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  base: "/app", // <-- specify base
  input: "/src/main.tsx",
  plugins: [react()],
  build: {
    manifest: true,
  },
})

```

Adjust static asset paths with vite_public.
```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="{{ vite_public "/vite.svg" }}" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + React + TS</title>
    {{ vite_react_refresh }} <!-- for React only -->
    {{ vite "src/main.tsx" }}
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```
