package main

import "github.com/eriicafes/tmpl"

func init() {
	tmpl.Define(`Hello world 1`)
	tmpl.Define(`Hello world 2`) // should be ignored
}
