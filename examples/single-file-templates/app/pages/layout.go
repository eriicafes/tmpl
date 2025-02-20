package pages

import "github.com/eriicafes/tmpl"

type Layout struct {
	tmpl.Children
	Title string
}

func (l Layout) Tmpl() tmpl.Template {
	return tmpl.Associated(l.Base(), "pages/layout", l)
}

func init() {
	tmpl.Define(`
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="icon" type="image/svg+xml" href="/globe.svg">
    <title>{{ .Title }} | Tmpl Vite</title>
    {{ vite "app/main.ts" }}
    {{ vite_css "app/main.css" }}

    <script>
        // sync theme
        if (localStorage.getItem("theme") === "dark") {
            document.documentElement.classList.add("dark")
        }
    </script>
    {{ block "script" . }}{{ end }}
</head>
<body class="flex flex-col min-h-screen [&>main]:flex-1 dark:bg-black dark:text-white">
    <header class="border-b border-zinc-200 dark:border-zinc-800">
        <div class="max-w-5xl mx-auto px-4 py-6 flex items-center justify-between">
            <a href="/" class="flex items-center gap-1">
                <img src="{{ vite_public "/globe.svg" }}" alt="Globe" class="size-4">
                <h1 class="text-sm tracking-wide font-semibold dark:text-zinc-400">Tmpl Vite</h1>
            </a>
            <button data-theme-toggle class="cursor-pointer hover:bg-zinc-100 dark:hover:bg-zinc-800 rounded-sm px-2 py-1 text-sm font-medium flex items-center gap-1">
                {{ template "components/icons/sun" map "class" "size-4" }}
                Toggle theme
            </button>
        </div>
    </header>

    {{ slot .Children }}

    <footer class="px-4 py-6 border-t bg-zinc-100 border-zinc-200 dark:bg-zinc-900 dark:border-zinc-800">
        <p class="text-center text-zinc-600 dark:text-zinc-400 text-xs font-medium">Go Tmpl + Vite + Tailwind</p>
    </footer>
</body>
</html>	
`)
}
