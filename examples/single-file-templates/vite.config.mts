import { defineConfig } from "vite"
import { globSync } from "glob"
import tailwindcss from "@tailwindcss/vite"

export default defineConfig({
    plugins: [tailwindcss()],
    build: {
        manifest: true,    
        rollupOptions: {
            input: globSync("./app/**/*.{ts,css}")
        }
    }
})
