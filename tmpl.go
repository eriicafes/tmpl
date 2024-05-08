package tmpl

import (
	"html/template"
	"io"
	"io/fs"
	"os"
)

// Template is the interface that enables types as templates.
//
// Template returns a template name and template data.
// The returned template data is directly available to the template.
type Template interface {
	Template() (name string, data any)
}

// AssociatedTemplate is the interface that enables types as associated templates.
// An associated template is any template that is parsed while loading a template.
// This includes all define blocks, layouts, autoloads and the loaded template itself.
//
// AssociatedTemplate returns a base template name, an associated template name and template data.
// The returned template data is directly available to the associated template.
type AssociatedTemplate interface {
	AssociatedTemplate() (base string, name string, data any)
}

// Templates is a map[string]*template.Template holding all loaded templates.
//
// If a template is not explicitly loaded by name, it is executed from the autoload template.
type Templates map[string]*template.Template

// Render executes a Template tp and writes the output to w.
func (t Templates) Render(w io.Writer, tp Template) error {
	name, data := tp.Template()
	tname := name
	if _, ok := t[tname]; !ok {
		tname = "autoload"
	}
	return t[tname].ExecuteTemplate(w, name, data)
}

// RenderAssociated executes the associated template with the given name for a Template tp and writes the output to w.
func (t Templates) RenderAssociated(w io.Writer, atp AssociatedTemplate) error {
	bname, name, data := atp.AssociatedTemplate()
	if _, ok := t[bname]; !ok {
		bname = "autoload"
	}
	return t[bname].ExecuteTemplate(w, name, data)
}

type templatesParser struct {
	fsys           fs.FS
	ext            string
	layoutDir      string
	layoutFilename string
	autoload       []string
	onLoadFn       func(string, *template.Template)
	templates      Templates
	loadErr        error
}

// New initializes a n new templates parser from any os dir.
func New(dir string) *templatesParser {
	return NewFS(os.DirFS(dir))
}

// NewFS initializes a new templates parser from any fs.FS.
func NewFS(fsys fs.FS) *templatesParser {
	return &templatesParser{
		fsys:           fsys,
		ext:            "html",
		layoutFilename: "layout",
		templates: Templates{
			"autoload": template.New("autoload").Funcs(funcMap),
		},
	}
}

// Clone clones a template parser with all it's loaded templates and configuration.
//
// Clone creates a copy of the template parser, so further calls to load templates
// or modify configurations will apply to the copy but not the original.
//
// Clone can be used to preapre templates with similar base autoloads or configurations.
//
// Clone returns an error if cloning any of the templates returns an error.
func (t *templatesParser) Clone() (*templatesParser, error) {
	var err error
	templates := make(Templates, len(t.templates))
	for k, v := range t.templates {
		if templates[k], err = v.Clone(); err != nil {
			return nil, err
		}
	}
	return &templatesParser{
		fsys:           t.fsys,
		ext:            t.ext,
		layoutDir:      t.layoutDir,
		layoutFilename: t.layoutFilename,
		autoload:       append(make([]string, 0, len(t.autoload)), t.autoload...),
		onLoadFn:       t.onLoadFn,
		templates:      templates,
		loadErr:        t.loadErr,
	}, nil
}

// MustClone clones a template parser with all it's loaded templates and configuration.
//
// MustClone creates a copy of the template parser, so further calls to load templates
// or modify configurations will apply to the copy but not the original.
//
// MustClone can be used to preapre templates with similar base autoloads or configurations.
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
// Default is html.
func (t *templatesParser) SetExt(ext string) *templatesParser {
	t.ext = ext
	return t
}

// SetExt sets the directory to start searching for layout files.
// Default is dir.
func (t *templatesParser) SetLayoutDir(dir string) *templatesParser {
	t.layoutDir = dir
	return t
}

// SetExt sets the filename of layout files.
// Default is layout.
//
// ie. if extension is html and layout filename is _layout,
// then any _layout.html file is a layout file.
func (t *templatesParser) SetLayoutFilename(filename string) *templatesParser {
	t.layoutFilename = filename
	return t
}

// OnLoad applies f to all loaded templates.
// f is called for each template before the template is parsed.
//
// OnLoad can be used to configure templates, for example, add templates funcs.
func (t *templatesParser) OnLoad(f func(name string, t *template.Template)) *templatesParser {
	t.onLoadFn = f
	t.onLoadFn("autoload", t.templates["autoload"])
	return t
}

// Autoload loads all templates in dirs.
// Autoloaded templates are available in all templates.
//
// Autoload can be used to load common templates like components.
func (t *templatesParser) Autoload(dirs ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(dirs) == 0 {
		return t
	}
	autoloadFilenames := walkFiles(t.fsys, t.ext, dirs, false)
	autoloadTmpl := t.templates["autoload"]
	_, err := parseFiles(t.fsys, autoloadTmpl, autoloadFilenames)
	if err != nil {
		t.loadErr = err
		return t
	}
	t.autoload = append(t.autoload, autoloadFilenames...)
	return t
}

// AutoloadShallow loads all templates in dirs omitting sub directories.
// Autoloaded templates are available in all templates.
//
// AutoloadShallow can be used to load common templates like components.
func (t *templatesParser) AutoloadShallow(dirs ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(dirs) == 0 {
		return t
	}
	autoloadFilenames := walkFiles(t.fsys, t.ext, dirs, true)
	autoloadTmpl := t.templates["autoload"]
	_, err := parseFiles(t.fsys, autoloadTmpl, autoloadFilenames)
	if err != nil {
		t.loadErr = err
		return t
	}
	t.autoload = append(t.autoload, autoloadFilenames...)
	return t
}

// Load loads a list of templates files.
// The template name is the last template filename.
// Templates must be loaded before they are used within another template
// hence the order with which they are loaded matters.
//
// example: when header is used within pages/dashboard
//
// tp.Load("header", "pages/dashboard")
//
// Prefer loading commonly used templates with Autoload.
func (t *templatesParser) Load(files ...string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	if len(files) == 0 {
		return t
	}
	name := files[len(files)-1]
	filenames := make([]string, 0, len(files))
	for _, filename := range files {
		filenames = append(filenames, filename+"."+t.ext)
	}
	err := t.load(name, filenames)
	if err != nil {
		t.loadErr = err
		return t
	}
	return t
}

// LoadWithLayouts loads all templates and their layout templates in a directory.
// The template name is the last template filename.
func (t *templatesParser) LoadWithLayouts(dir string) *templatesParser {
	if t.loadErr != nil {
		return t
	}
	groups := walkFilesWithLayout(t.fsys, t.ext, t.layoutDir, t.layoutFilename, dir)
	for name, files := range groups {
		err := t.load(name, files)
		if err != nil {
			t.loadErr = err
			break
		}
	}
	return t
}

// load creates a new template with name and parses autoload templates and all templates in files.
func (t *templatesParser) load(name string, files []string) error {
	if len(files) == 0 {
		return nil
	}
	tmpl := template.New(name).Funcs(funcMap)
	if t.onLoadFn != nil {
		t.onLoadFn(name, tmpl)
	}
	tmpl, err := parseFiles(t.fsys, tmpl, t.autoload)
	if err != nil {
		return err
	}
	tmpl, err = parseFiles(t.fsys, tmpl, files)
	if err != nil {
		return err
	}
	t.templates[name] = tmpl
	return nil
}

// Parse parses and returns Templates.
//
// Parse returns an error if parsing any of the templates returns an error.
func (t *templatesParser) Parse() (Templates, error) {
	if t.loadErr != nil {
		return nil, t.loadErr
	}
	return t.templates, nil
}

// MustParse parses and returns Templates.
//
// MustParse panics if parsing any of the templates returns an error.
func (t *templatesParser) MustParse() Templates {
	if t.loadErr != nil {
		panic(t.loadErr)
	}
	return t.templates
}
