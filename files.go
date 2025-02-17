package tmpl

import (
	"html/template"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// Define defines the template content for a go file.
//
// Define must use backticks and must be executed exactly once in the init function of the go file.
// The content is interpreted as-is as the code is not actually executed.
//
// Define is a noop.
func Define(content string) {}

func extractGoFileContent(s string) string {
	// match content within tmpl.Content(`content`)
	// account for syntaxt hughlighting comments like tmpl.Content( /* html */ `content`)
	re := regexp.MustCompile(`tmpl\.Content\(\s*(?:\/\*.*?\*\/\s*)?` + "`([^`]*)`" + `\)`)
	matches := re.FindStringSubmatch(s)

	if len(matches) > 1 {
		// trim and return extracted content inside backticks
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// parseFiles parses template files into t.
//
// Repeated template names are overriden.
func parseFiles(fsys fs.FS, t *template.Template, ext string, files []string) error {
	for _, name := range files {
		filename := name
		if ext != "" {
			filename = name + "." + ext
		}
		b, err := fs.ReadFile(fsys, filename)
		if err != nil {
			return err
		}
		tmpl := t.New(name)
		text := string(b)
		if ext == "go" {
			text = extractGoFileContent(text)
		}
		_, err = tmpl.Parse(text)
		if err != nil {
			return err
		}
	}
	return nil
}

// walkFiles walks dirs and returns a slice of filenames (without extension) that matches the file extension ext.
func walkFiles(fsys fs.FS, ext string, dirs []string) []string {
	var files []string
	for _, dir := range dirs {
		fs.WalkDir(fsys, dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return err
			}
			pathWithoutExt := strings.TrimSuffix(path, "."+ext)
			// if pathWithoutExt remained unchanged then path does not have ext
			if ext != "" && path == pathWithoutExt {
				return err
			}
			files = append(files, pathWithoutExt)
			return err
		})
	}
	return files
}

// walkFilesWithLayout walks a directory and for each filename that matches the file extension ext,
// returns a slice of all layout filenames (without extension) in parent directories and the matched filename (without extension).
func walkFilesWithLayout(fsys fs.FS, ext string, layoutFilename string, dir string) map[string][]string {
	groups := make(map[string][]string)
	layouts := make([]string, 0)
	fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return err
		}
		pathWithoutExt := strings.TrimSuffix(path, "."+ext)
		// if pathWithoutExt remained unchanged then path does not have ext
		if ext != "" && path == pathWithoutExt {
			return err
		}
		_, filename := filepath.Split(pathWithoutExt)
		if filename == layoutFilename {
			layouts = append(layouts, pathWithoutExt)
		} else if dir == "." || strings.HasPrefix(pathWithoutExt, dir) {
			groups[pathWithoutExt] = []string{pathWithoutExt}
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
			// no need to check deeper layout files
			if layoutDir == fileDir {
				break
			}
		}
		groups[name] = append(files, groups[name]...)
	}
	return groups
}
