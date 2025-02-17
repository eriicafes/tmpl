package tmpl

import (
	"bytes"
	"html/template"
	"maps"
	"os"
	"slices"
	"testing"
	"testing/fstest"
)

func TestWalkFiles(t *testing.T) {
	fs := fstest.MapFS{
		"index.html":         {},
		"index.tmpl":         {},
		"index":              {},
		"test/index.html":    {},
		"test/index.tmpl":    {},
		"test/index":         {},
		"app/index.html":     {},
		"app/home.html":      {},
		"app/dashboard.tmpl": {},
		"app/dashboard":      {},
		"auth/index.tmpl":    {},
		"auth/index":         {},
		"auth/login.html":    {},
		"auth/register.html": {},
	}

	// walk root
	expected := []string{
		"index",
		"test/index",
		"app/index",
		"app/home",
		"auth/login",
		"auth/register",
	}
	got := walkFiles(fs, "html", []string{"."})
	slices.Sort(expected)
	slices.Sort(got)
	if !slices.Equal(expected, got) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}

	// walk sub dirs
	expected = []string{
		"app/index",
		"app/home",
		"auth/login",
		"auth/register",
	}
	got = walkFiles(fs, "html", []string{"auth", "app"})
	slices.Sort(expected)
	slices.Sort(got)
	if !slices.Equal(expected, got) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestWalkFilesWithLayout(t *testing.T) {
	fs := fstest.MapFS{
		"index.html":              {},
		"index.tmpl":              {},
		"index":                   {},
		"layout.html":             {},
		"test/layout.html":        {},
		"test/index.tmpl":         {},
		"test/index":              {},
		"app/layout.html":         {},
		"app/index.html":          {},
		"app/dashboard.html":      {},
		"app/dashboard.tmpl":      {},
		"app/dashboard":           {},
		"app/account/layout.html": {},
		"app/account/index.html":  {},
		"auth/index.tmpl":         {},
		"auth/index":              {},
		"auth/login.html":         {},
		"auth/register.html":      {},
	}

	// walk root
	expected := map[string][]string{
		"index":             {"layout", "index"},
		"app/index":         {"layout", "app/layout", "app/index"},
		"app/dashboard":     {"layout", "app/layout", "app/dashboard"},
		"app/account/index": {"layout", "app/layout", "app/account/layout", "app/account/index"},
		"auth/login":        {"layout", "auth/login"},
		"auth/register":     {"layout", "auth/register"},
	}
	got := walkFilesWithLayout(fs, "html", "layout", ".")
	if !maps.EqualFunc(expected, got, slices.Equal) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}

	// walk sub dir
	expected = map[string][]string{
		"app/index":         {"layout", "app/layout", "app/index"},
		"app/dashboard":     {"layout", "app/layout", "app/dashboard"},
		"app/account/index": {"layout", "app/layout", "app/account/layout", "app/account/index"},
	}
	got = walkFilesWithLayout(fs, "html", "layout", "app")
	if !maps.EqualFunc(expected, got, slices.Equal) {
		t.Errorf("expected: %v, got: %v", expected, got)
	}
}

func TestGoTemplateFile(t *testing.T) {
	fs := os.DirFS("testdata/go-file-templates")

	tests := []struct {
		file     string
		expected string
	}{
		{"single", "Hello world"},
		{"double", "Hello world 1"},
		{"none", ""},
		{"quotes", ""},
		{"quotes-and-backticks", "Hello world 2"},
		{"comment", "Hello world"},
	}

	var files []string
	for _, test := range tests {
		files = append(files, test.file)
	}

	tp := template.New("templates")
	err := parseFiles(fs, tp, "go", files)
	if err != nil {
		t.Error(err)
	}

	buf := new(bytes.Buffer)
	for _, test := range tests {
		err := tp.ExecuteTemplate(buf, test.file, nil)
		if err != nil {
			t.Error(err)
		}
		if buf.String() != test.expected {
			t.Errorf("expected: %s %q, got: %q", test.file, test.expected, buf.String())
		}
		buf.Reset()
	}
}
