package tmpl

import (
	"fmt"
	"html/template"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
)

// parseFiles parses template files into t.
//
// Repeated template names are overriden.
func parseFiles(fsys fs.FS, t *template.Template, ext string, filenames []string) (*template.Template, error) {
	for _, filename := range filenames {
		b, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return nil, err
		}
		name := strings.TrimSuffix(filename, "."+ext)
		var tmpl *template.Template
		if name == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(name)
		}
		_, err = tmpl.Parse(string(b))
		if err != nil {
			return nil, err
		}
	}
	return t, nil
}

// walkFiles walks dirs and returns a slice of filenames that matches the file extension ext.
//
// If shallow is true, sub directories will not be walked otherwise walfFiles is recursive.
func walkFiles(fsys fs.FS, ext string, dirs []string, shallow bool) []string {
	var files []string
	for _, dir := range dirs {
		fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if shallow && d.IsDir() && path != dir {
				return fs.SkipDir
			}
			if d.IsDir() {
				return err
			}
			if ext != "" && !strings.HasSuffix(d.Name(), "."+ext) {
				return err
			}
			files = append(files, path)
			return err
		})
	}
	return files
}

// walkFilesWithLayout walks a directory and for each filename that matches the file extension ext,
// returns a slice of all layouts filenames in parent directories matching layoutFilename and the matched filename.
//
// If layoutRoot is not provided only layout filenames from dir and it's sub directories will be matched.
// Use layoutRoot to start walking at a higher up directory than dir if there are layouts outside of dir.
func walkFilesWithLayout(fsys fs.FS, ext string, layoutRoot string, layoutFilename string, dir string) map[string][]string {
	groups := make(map[string][]string)
	layouts := make([]string, 0)
	if layoutRoot == "" {
		layoutRoot = dir
	}
	fs.WalkDir(fsys, layoutRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return err
		}
		pathWithoutExt := strings.TrimSuffix(path, "."+ext)
		if ext != "" && path == pathWithoutExt {
			return err
		}
		fmt.Println("walked path:", pathWithoutExt)
		_, filename := filepath.Split(pathWithoutExt)
		if filename == layoutFilename {
			layouts = append(layouts, path)
		} else if strings.HasPrefix(pathWithoutExt, dir) {
			groups[pathWithoutExt] = []string{path}
		}
		return err
	})

	if len(layouts) < 1 {
		return groups
	}
	slices.SortFunc(layouts, func(a, b string) int {
		return len(a) - len(b)
	})
	for name := range groups {
		files := []string{}
		fileDir, _ := filepath.Split(name)
		for _, layout := range layouts {
			layoutDir, _ := filepath.Split(layout)
			if strings.HasPrefix(fileDir, layoutDir) {
				files = append(files, layout)
			}
			if layoutDir == fileDir {
				break
			}
		}
		if len(files) > 0 {
			groups[name] = append(files, groups[name]...)
		}
	}
	return groups
}
