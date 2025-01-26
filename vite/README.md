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
	v, err := vite.New(vite.Config{Dev: true}) // <-- create vite instance
	if err != nil {
		panic(err)
	}
	templates := tmpl.New(os.DirFS("templates")).
		Funcs(v.Funcs()). // <-- register vite template funcs
		LoadTree("pages").
		MustParse()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := templates.Render(w, tmpl.Tmpl("pages/index"))
		if err != nil {
			fmt.Println(err)
		}
	})
	http.Handle("/", v.ServePublic(handler)) // <-- wrap handler with vite middleware
	http.ListenAndServe(":8000", nil)
}
```

#### 2. Update vite config.

Enable vite manifest and change vite entry point.

```ts
// vite.config.ts
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    manifest: true, // <-- enable vite manifest
    rollupOptions: {
      input: "src/main.tsx" // <-- change entry point
    },
  },
})
```

#### 3. Return html document with the vite tags.

Render the vite tags in your template html head.

`{{ vite "path/to/input.js" }}`

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

vite returns the required vite tags. For each of the inputs it returns a script tag with the src set to the asset path.

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

#### vite_asset

vite_asset references static assets relative to their path in source code.
In development it resolves the path on the vite development server.
In production it resolves to the vite build output.

```html
<!doctype html>
<html lang="en">
  <head>...</head>
  <body>
    <!-- executing this -->
    <script type="module" src="{{ vite_asset "src/app.ts" }}"></script>

    <!-- returns this in development -->
    <script type="module" src="http://localhost:5173/src/app.ts"></script>

    <!-- returns this in production -->
    <script type="module" src="/assets/app.js"></script>
  </body>
</html>
```

#### vite_react_refresh

vite_react_refresh returns the react refresh preamble.
If you are using React with `@vitejs/plugin-react`, you'll need to add this before the vite tags.

```html
<!doctype html>
<html lang="en">
  <head>...</head>
  <body>
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
  </body>
</html>
```

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
		err := templates.Render(w, tmpl.Tmpl("pages/index"))
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
  plugins: [react()],
  build: {
    manifest: true,
    rollupOptions: {
      input: "src/main.tsx"
    },
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