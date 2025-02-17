package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"tmpl-example/app/pages"
	profile_pages "tmpl-example/app/pages/profile"

	"github.com/eriicafes/tmpl"
	"github.com/eriicafes/tmpl/vite"
)

func main() {
	dev := flag.Bool("dev", false, "development mode")
	flag.Parse()

	v, _ := vite.New(vite.Config{
		Dev: *dev,
	})
	fs := os.DirFS("app")
	tp := tmpl.New(fs).Funcs(v.Funcs()).Autoload("components").LoadTree("pages").MustParse()

	http.HandleFunc("GET /{$}", func(w http.ResponseWriter, r *http.Request) {
		var name string
		var count int
		if cookie, err := r.Cookie("Name-State"); err == nil {
			name = cookie.Value
		}
		if cookie, err := r.Cookie("Count-State"); err == nil {
			count, _ = strconv.Atoi(cookie.Value)
		}
		err := tp.Render(w, pages.Index{
			Layout: pages.Layout{
				Title: "Welcome",
			},
			Name:  name,
			Count: count,
		})
		if err != nil {
			fmt.Println(err)
		}
	})
	http.HandleFunc("GET /profile", func(w http.ResponseWriter, r *http.Request) {
		var name string
		if cookie, err := r.Cookie("Name-State"); err == nil {
			name = cookie.Value
		}
		err := tp.Render(w, profile_pages.Index{
			Name: name,
		})
		if err != nil {
			fmt.Println(err)
		}
	})
	http.HandleFunc("POST /profile", func(w http.ResponseWriter, r *http.Request) {
		var name = r.PostFormValue("name")
		http.SetCookie(w, &http.Cookie{
			Name:  "Name-State",
			Value: name,
			Path:  "/",
		})
		http.Redirect(w, r, "/", http.StatusFound)
	})
	http.Handle("/", v.ServePublic(http.NotFoundHandler()))
	err := http.ListenAndServe(":8000", nil)
	panic(err)
}
