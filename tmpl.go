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
	return &templatesParser{
		fsys:           fsys,
		ext:            "html",
		layoutFilename: "layout",
		templates: Templates{
			"<root>": template.New("<root>").Funcs(funcMap),
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
	var err error
	for k, v := range t.templates {
		if templates[k], err = v.Clone(); err != nil {
			return nil, err
		}
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

// Load creates a new template and parses the template definitions from the named files.
// The created template clones the root template so all autoloaded templates are available.
//
// The template's name will have the filepath of the first file.
// Similarly, the remaining files will be associated templates named with their filepath.
//
// If an error occurs, it will be returned when calling Parse and further calls to Load will be a noop.
//
// Templates file names are their filepath without the extension, this act as a namespace to avoid name collisions.
// The file extension can be configured using SetExt, the default is "html".
//
// For instance, Load("a/foo", "b/foo") creates the template named "a/foo", and an associated template to "a/foo" named "b/foo" is also created.
func (t *templatesParser) Load(files ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(files) == 0 {
		return t
	}
	t.loadErr = t.load(files[0], files)
	return t
}

// LoadTree creates all templates in a directory loading all their respective layout templates.
//
// If a layout root is not provided only layout files from dir and it's sub directories will be matched.
// Use SetLayoutRoot to start walking at a higher up directory than dir if there are layouts outside of dir.
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
