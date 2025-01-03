## Vite Integration

Vite is a powerful frontend build tool which provides a handful of tools including bundling of static assets, CSS, JS and TypeScript source code for development and production.

Tmpl provides first-party support for [Vite](https://vite.dev). In development Tmpl proxies requests to static assets to the vite development server while in production it serves the vite output directory.

Configure vite in 3 easy steps:

#### 1. Create vite instance, add templates funcs and setup middleware.

Vite funcs include the following:
- vite
- vite_public
- vite_asset
- vite_react_refresh

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
	v, err := vite.New(vite.Config{Dev: true}) // create vite instance
	if err != nil {
		panic(err)
	}
	templates := tmpl.New(os.DirFS("templates")).
		Funcs(v.Funcs()). // register vite template funcs
		LoadTree("pages").
		MustParse()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := templates.Renderer()

		err := tr.Render(w, tmpl.Tmpl("pages/index"))
		if err != nil {
			fmt.Println(err)
		}
	})
	http.Handle("/", v.ServePublic(handler)) // wrap handler with vite middleware
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
    manifest: true, // enable vite manifest
    rollupOptions: {
      input: "src/main.tsx" // change entry point
    },
  },
})
```

#### 3. Return html document with the vite tags.

Render the vite tags in your template html head.

{{ vite "path/to/input.js" }}

Additionally for react using @vitejs/plugin-react render the following tag before the vite tags.

{{ vite_react_refresh }}

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + React + TS</title>
    {{ vite_react_refresh }} <!-- for react only -->
    {{ vite "src/main.tsx" }}
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```

### Vite Funcs
Use vite_public or vite_static to reference static files or src files respectively. In development requests are proxied to the vite development server. In production it resolves to the vite build output.

#### vite_public

Use vite_public to reference static assets using their absolute path from the vite publicDir.

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + TS</title>
    {{ vite "src/main.ts" }}
  </head>
  <body>
    <img src="{{ vite_public "images/logo.png" }}" width="200" height="200" />
  </body>
</html>
```

#### vite_asset

Use vite_asset to reference static assets using their absolute path source code.

```html
<!doctype html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <link rel="icon" type="image/svg+xml" href="/vite.svg" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>Vite + TS</title>
    {{ vite "src/main.ts" }}
  </head>
  <body>
    <script type="module" src="{{ vite_asset "/src/app.ts" }}"></script>
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
	v, err := vite.New(vite.Config{Dev: true, Base: "/app"}) // specify base
	if err != nil {
		panic(err)
	}
	templates := tmpl.New(os.DirFS("templates")).
		Funcs(v.Funcs()).
		LoadTree("pages").
		MustParse()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tr := templates.Renderer()

		err := tr.Render(w, tmpl.Tmpl("pages/index"))
		if err != nil {
			fmt.Println(err)
		}
	})
	http.Handle("/app/", v.ServePublic(handler)) // specify base
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
  base: "/app", // specify base
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
    {{ vite_react_refresh }} <!-- for react projects only -->
    {{ vite "src/main.tsx" }}
  </head>
  <body>
    <div id="root"></div>
  </body>
</html>
```