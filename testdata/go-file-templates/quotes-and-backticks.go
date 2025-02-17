package main

import "github.com/eriicafes/tmpl"

func init() {
	tmpl.Define("Hello world 1") // should be ignored
	tmpl.Define(`Hello world 2`)
}
