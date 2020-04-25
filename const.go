package gutowire

import (
	"html/template"
)

const (
	filePrefix  = "autowire"
	wireTag     = "@autowire"
	setTemplate = `// Code generated by go-autowire. DO NOT EDIT.

package {{ .Package }}

import (
	"github.com/google/wire"
)

var {{ .SetName }} = wire.NewSet({{ range $Item := .Items}} 
	{{ $Item }},
    {{ end }}
)
`
)

var setTemp = template.Must(template.New("").Parse(setTemplate))

type (
	wireSet struct {
		Package string
		Items   []template.HTML
		SetName string
	}

	opt struct {
		searchPath string
		pkg        string
		genPath    string
	}

	Option func(*opt)

	searcher struct {
		sets       []string
		genPath    string
		pkg        string
		elementMap map[string]map[string]element
		options    []Option
		modBase    string
	}

	element struct {
		name        string
		constructor string
		field       []string
		implements  []string
		pkg         string
		pkgPath     string
		typ         uint
	}

	tmpDecl struct {
		docs   string
		name   string
		isFunc bool
	}
)

func WithPkg(pkg string) Option {
	return func(o *opt) {
		o.pkg = pkg
	}
}

func WithSearchPath(path string) Option {
	return func(o *opt) {
		o.searchPath = path
	}
}