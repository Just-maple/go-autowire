package gutowire

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/iancoleman/strcase"
)

func SearchAllPath(file string, genPath string, pkg string, opts ...Option) (err error) {
	sc, ok := searcherStore[file]
	if ok {
		sc.genPath = genPath
		sc.pkg = pkg
		return writeGen(sc)
	}
	modBaser, err := getModBase()
	if err != nil {
		return
	}
	pkg = strings.Replace(pkg, "-", "_", -1)
	sc = &searcher{
		genPath:    genPath,
		pkg:        pkg,
		options:    opts,
		elementMap: make(map[string]map[string]element),
		modBase:    modBaser,
	}
	err = sc.SearchAllPath(file)
	if err != nil {
		return
	}
	searcherStore[file] = sc
	log.Printf("analysis autowire complete")
	return writeGen(sc)
}

func (sc *searcher) SearchAllPath(file string) (err error) {
	err = filepath.Walk(file, func(path string, f os.FileInfo, err error) error {
		fn := f.Name()
		if f.IsDir() && (fn == "vendor" || fn == "testdata") {
			return filepath.SkipDir
		}
		if !f.IsDir() && strings.HasSuffix(fn, ".go") && !strings.HasSuffix(fn, "_test.go") {
			err = sc.searchWire(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	return
}

func (sc *searcher) searchWire(file string) (err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(wireTag)) {
		return
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "", data, parser.ParseComments)
	if err != nil {
		return
	}
	var tmpDecls []tmpDecl
	for _, decl := range f.Decls {
		if d, ok := decl.(*ast.GenDecl); ok {
			if !(d.Tok.String() == "type") {
				continue
			}
			if len(d.Specs) == 1 && strings.Contains(d.Doc.Text(), wireTag) {
				id, ok := d.Specs[0].(*ast.TypeSpec)
				if !ok {
					continue
				}
				tmpDecls = append(tmpDecls, tmpDecl{
					docs:     d.Doc.Text(),
					name:     id.Name.Name,
					isFunc:   false,
					typeSpec: id,
				})
				continue
			}
			for _, sp := range d.Specs {
				id, ok := sp.(*ast.TypeSpec)
				if !(ok && strings.Contains(id.Doc.Text(), wireTag)) {
					continue
				}
				tmpDecls = append(tmpDecls, tmpDecl{
					docs:     id.Doc.Text(),
					name:     id.Name.Name,
					isFunc:   false,
					typeSpec: id,
				})
				continue

			}
		} else if f, ok := decl.(*ast.FuncDecl); ok && strings.Contains(f.Doc.Text(), wireTag) {
			tmpDecls = append(tmpDecls, tmpDecl{
				docs:   f.Doc.Text(),
				name:   f.Name.Name,
				isFunc: true,
			})
		}
	}
	implementMap := getImplement(f)
	for _, decl := range tmpDecls {
		lines := strings.Split(decl.docs, "\n")
		for _, c := range lines {
			sc.analysisWireTag(c, file, &decl, f, implementMap)
		}
	}
	return
}

func (sc *searcher) getPkgPath(filePath string) (pkgPath string) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	dir := getGoModDir()
	if len(abs) < len(dir) {
		return
	}
	pkgPath = filepath.ToSlash(filepath.Dir(filepath.Join(sc.modBase, abs[len(dir):])))
	return
}

func (sc *searcher) analysisWireTag(rawTag, filePath string, decl *tmpDecl, f *ast.File, implementMap map[string]string) {
	var (
		isFunc  = decl.isFunc
		name    = decl.name
		pkgPath = sc.getPkgPath(filePath)
		tag     = strings.TrimSpace(rawTag)
	)

	if !strings.HasPrefix(tag, wireTag) {
		return
	}
	tagStr := tag[len(wireTag):]
	if !(strings.HasPrefix(tagStr, "(") && strings.HasSuffix(tagStr, ")")) {
		return
	}
	options := map[string]string{}
	// todo:support more
	// @autowire(interface,interface,set=setName,field=*)
	tagStr = tagStr[1 : len(tagStr)-1]
	sp := strings.Split(tagStr, ",")
	for _, s := range sp {
		s = strings.TrimSpace(s)
		if len(s) == 0 {
			continue
		}
		spo := strings.Split(s, "=")
		var v string
		if len(spo) > 1 {
			v = spo[1]
		}
		options[spo[0]] = v
	}
	e := element{
		name:        name,
		constructor: "",
		pkg:         f.Name.Name,
		pkgPath:     pkgPath,
	}
	if isFunc {
		e.constructor = name
	} else {
		for _, cn := range []string{"Init", "New"} {
			if ct, ok := f.Scope.Objects[cn+name]; ok && ct.Kind == ast.Fun {
				e.constructor = cn + name
				break
			}
		}
	}

	var setName string
	if len(options["set"]) == 0 {
		setName = "unknown"
		if sc.elementMap[setName] == nil {
			sc.elementMap[setName] = make(map[string]element)
		}
		sc.elementMap[setName][path.Join(pkgPath, name)] = e
		return
	} else {
		setName = strcase.ToLowerCamel(options["set"])
	}

	if sc.elementMap[setName] == nil {
		sc.elementMap[setName] = make(map[string]element)
	}
	defer func() {
		log.Printf("%sSet : %s\n", strcase.ToLowerCamel(setName), e.pkg+"."+e.name)
		sc.elementMap[setName][path.Join(pkgPath, name)] = e
	}()

	for key, value := range options {
		switch key {
		case "set":
			continue
		case "field":
			e.field = append(e.field, value)
		case "new":
			e.constructor = value
		default:
			e.implements = append(e.implements, key)
		}
	}
	if len(implementMap[name]) > 0 {
		insertIfUnExist(implementMap[name], &e.implements)
	}
}

func getImplement(f *ast.File) (ret map[string]string) {
	ret = make(map[string]string)
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range gd.Specs {
			vs, ok := sp.(*ast.ValueSpec)
			if !ok || vs.Names[0].Name != "_" || vs.Type == nil || len(vs.Values) != 1 {
				continue
			}
			var id *ast.Ident
			switch t := vs.Values[0].(type) {
			case *ast.CompositeLit:
				id, ok = t.Type.(*ast.Ident)
				if !ok {
					continue
				}
			case *ast.UnaryExpr:
				if t.Op != token.AND {
					continue
				}
				cl, ok := t.X.(*ast.CompositeLit)
				if !ok {
					continue
				}
				id, ok = cl.Type.(*ast.Ident)
				if !ok {
					continue
				}
			default:
				continue
			}
			imp, ok := vs.Type.(*ast.Ident)
			if !ok {
				continue
			}
			ret[id.Name] = imp.Name
		}
	}
	return
}

func insertIfUnExist(i string, sl *[]string) {
	for _, s := range *sl {
		if s == i {
			return
		}
	}
	*sl = append(*sl, i)
}

func writeGen(sc *searcher) (err error) {
	if len(sc.elementMap) == 0 {
		return
	}
	return sc.write()
}
