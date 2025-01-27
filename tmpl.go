package tmpl

import (
	"html/template"
	"io/fs"
)

// Templates is a map[string]*template.Template holding all loaded templates.
//
// If a template is not explicitly loaded by name, it is executed from the autoload template.
type Templates map[string]*template.Template

type templatesParser struct {
	fsys           fs.FS
	ext            string
	layoutFilename string
	templates      Templates
	loadErr        error
	onLoadFn       func(string, *template.Template)
}

// New initializes a new templates parser from any fs.FS.
func New(fsys fs.FS) *templatesParser {
	root := template.New("<root>")
	return &templatesParser{
		fsys:           fsys,
		ext:            "html",
		layoutFilename: "layout",
		templates: Templates{
			"<root>": root.Funcs(funcMap).Funcs(contextFuncMap(root)),
		},
	}
}

// Clone clones a template parser with all it's loaded templates.
//
// Clone creates a copy of the template parser, so further calls to load templates
// will apply to the copy but not the original.
//
// Clone returns an error if cloning any of the templates returns an error.
func (t *templatesParser) Clone() (*templatesParser, error) {
	templates := make(Templates, len(t.templates))
	for k, v := range t.templates {
		clone, err := v.Clone()
		if err != nil {
			return nil, err
		}
		templates[k] = clone.Funcs(contextFuncMap(clone))
	}
	return &templatesParser{
		fsys:           t.fsys,
		ext:            t.ext,
		layoutFilename: t.layoutFilename,
		templates:      templates,
		loadErr:        t.loadErr,
		onLoadFn:       t.onLoadFn,
	}, nil
}

// MustClone clones a template parser with all it's loaded templates and configuration.
//
// MustClone creates a copy of the template parser, so further calls to load templates
// will apply to the copy but not the original.
//
// MustClone panics if cloning any of the templates returns an error.
func (t *templatesParser) MustClone() *templatesParser {
	tc, err := t.Clone()
	if err != nil {
		panic(err)
	}
	return tc
}

// SetExt sets the file extension of template files.
// Default is "html".
func (t *templatesParser) SetExt(ext string) *templatesParser {
	t.ext = ext
	return t
}

// SetLayoutFilename sets the filename of layout files.
// Default is "layout".
//
// ie. if extension is html and layout filename is _layout,
// then any _layout.html file is a layout file.
func (t *templatesParser) SetLayoutFilename(filename string) *templatesParser {
	t.layoutFilename = filename
	return t
}

// Funcs adds the func maps to the template's func map.
func (t *templatesParser) Funcs(funcMaps ...template.FuncMap) *templatesParser {
	for _, f := range funcMaps {
		t.templates["<root>"].Funcs(f)
	}
	return t
}

// OnLoad applies f to all loaded templates.
// f is called for each template before the template is parsed.
//
// OnLoad can be used to configure templates.
func (t *templatesParser) OnLoad(f func(name string, t *template.Template)) *templatesParser {
	t.onLoadFn = f
	return t
}

// Autoload loads all templates in dirs to the root template.
// The autoloaded templates are available in loaded templates as they clone the root template.
//
// Autoload can be used to load common templates like components.
func (t *templatesParser) Autoload(dirs ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(dirs) == 0 {
		return t
	}
	files := walkFiles(t.fsys, t.ext, dirs)
	t.loadErr = parseFiles(t.fsys, t.templates["<root>"], t.ext, files)
	return t
}

// Load loads a new template and parses the template definitions from the named files.
// The template is named after the last file and the other files will be associated templates.
//
// Template definitions in a file overrides template definitions in files to it's left
// hence the returned template is named after the rightmost file.
//
// If an error occurs, it will be returned when calling Parse and further calls to Load will be a noop.
//
// Templates file names are their filepath without the extension, this act as a namespace to avoid name collisions.
// The file extension can be configured using SetExt, the default is "html".
//
// For instance, Load("a/foo", "b/foo") loads the template named "b/foo" and an associated template named "a/foo".
func (t *templatesParser) Load(files ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(files) == 0 {
		return t
	}
	t.loadErr = t.load(files[len(files)-1], files)
	return t
}

// LoadTree loads all templates in a directory including all layout templates.
func (t *templatesParser) LoadTree(dir string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	groups := walkFilesWithLayout(t.fsys, t.ext, t.layoutFilename, dir)
	for name, files := range groups {
		if err := t.load(name, files); err != nil {
			t.loadErr = err
			break
		}
	}
	return t
}

// load clones the root template and parses the named files into the new template.
func (t *templatesParser) load(name string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	tmpl, err := t.templates["<root>"].Clone()
	if err != nil {
		return err
	}
	tmpl.Funcs(contextFuncMap(tmpl))
	if t.onLoadFn != nil {
		t.onLoadFn(name, tmpl)
	}
	err = parseFiles(t.fsys, tmpl, t.ext, files)
	if err != nil {
		return err
	}
	t.templates[name] = tmpl
	return nil
}

// Parse parses and returns Templates.
//
// Parse returns an error if loading any of the templates returned an error.
func (t *templatesParser) Parse() (Templates, error) {
	if t.loadErr != nil {
		return nil, t.loadErr
	}
	return t.templates, nil
}

// MustParse parses and returns Templates.
//
// MustParse panics if loading any of the templates returned an error.
func (t *templatesParser) MustParse() Templates {
	if t.loadErr != nil {
		panic(t.loadErr)
	}
	return t.templates
}
