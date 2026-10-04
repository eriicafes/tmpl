import { defineConfig } from "vite"
import { globSync } from "glob"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
    input: globSync("./app/**/*.{ts,css}"),
    plugins: [tailwindcss()],
    build: {
        manifest: true,
    }
})
