package gutowire

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/stoewer/go-strcase"
)

func RunAutoWireGen(genPath string, opts ...Option) (err error) {
	o := newGenOpt(genPath, opts...)
	file := o.searchPath
	pkg := o.pkg
	modBase, err := getModBase()
	if err != nil {
		return
	}
	pkg = strings.Replace(pkg, "-", "_", -1)
	sc := &autoWireSearcher{
		genPath:    genPath,
		pkg:        pkg,
		elementMap: make(map[string]map[string]element),
		modBase:    modBase,
		initWire:   o.initWire,
	}
	err = sc.SearchAllPath(file)
	if err != nil {
		return
	}
	log.Printf("analysis autowire complete")
	if len(sc.elementMap) == 0 {
		return
	}
	return sc.write()
}

func (sc *autoWireSearcher) SearchAllPath(file string) (err error) {
	return filepath.Walk(file, func(path string, f os.FileInfo, err error) error {
		fn := f.Name()
		if f.IsDir() && (fn == "vendor" || fn == "testdata") {
			return filepath.SkipDir
		}
		if f.IsDir() || !strings.HasSuffix(fn, ".go") || strings.HasSuffix(fn, "_test.go") {
			return nil
		}
		return sc.searchWire(path)
	})
}

func (sc *autoWireSearcher) searchWire(file string) (err error) {
	data, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	if !bytes.Contains(data, []byte(wireTag)) {
		return
	}

	parseFile, err := parser.ParseFile(token.NewFileSet(), "", data, parser.ParseComments)
	if err != nil {
		return
	}

	genPkgPath := fmt.Sprintf(`"%s"`, sc.getPkgPath(filepath.Join(sc.genPath, "...")))

	// to avoid import cycle
	for _, imp := range parseFile.Imports {
		if imp.Path.Value == genPkgPath {
			log.Printf("[warn] pacakge %s from [ %s ] ignore to avoid import cycle", parseFile.Name.Name, file)
			return
		}
	}

	var matchDecls []tmpDecl

	for _, decl := range parseFile.Decls {
		switch d := decl.(type) {
		case *ast.GenDecl:
			if !(d.Tok.String() == "type") {
				continue
			}
			/*
				@autowire()
				type Some struct{

				}
			*/
			if len(d.Specs) == 1 && strings.Contains(d.Doc.Text(), wireTag) {
				id, ok := d.Specs[0].(*ast.TypeSpec)
				if !ok {
					continue
				}
				matchDecls = append(matchDecls, tmpDecl{
					docs:     d.Doc.Text(),
					name:     id.Name.Name,
					isFunc:   false,
					typeSpec: id,
				})
				continue
			}
			/*
				type (
					@autowire()
					A struct{}

					@autowire()
					B struct{}
				)
			*/
			for _, sp := range d.Specs {
				id, ok := sp.(*ast.TypeSpec)
				if !(ok && strings.Contains(id.Doc.Text(), wireTag)) {
					continue
				}
				matchDecls = append(matchDecls, tmpDecl{
					docs:     id.Doc.Text(),
					name:     id.Name.Name,
					isFunc:   false,
					typeSpec: id,
				})
				continue

			}
		case *ast.FuncDecl:
			/*
				@autowire()
				func ConstructorForSomething() Something
			*/
			if !strings.Contains(d.Doc.Text(), wireTag) {
				continue
			}
			matchDecls = append(matchDecls, tmpDecl{
				docs:   d.Doc.Text(),
				name:   d.Name.Name,
				isFunc: true,
			})
		}
	}
	implementMap := getImplement(parseFile)
	for _, decl := range matchDecls {
		lines := strings.Split(decl.docs, "\n")
		for _, c := range lines {
			sc.analysisWireTag(strings.TrimSpace(c), file, &decl, parseFile, implementMap)
		}
	}
	return
}

func (sc *autoWireSearcher) getPkgPath(filePath string) (pkgPath string) {
	return getPkgPath(filePath, sc.modBase)
}

func (sc *autoWireSearcher) analysisWireTag(tag, filePath string, decl *tmpDecl, f *ast.File, implementMap map[string]string) {
	if !strings.HasPrefix(tag, wireTag) {
		return
	}

	var (
		itemFunc string

		isFunc  = decl.isFunc
		name    = decl.name
		pkgPath = sc.getPkgPath(filePath)
		tagStr  = tag[len(wireTag):]
	)

	if tagStr[0] == '.' {
		idx := strings.IndexRune(tagStr, '(')
		if idx == -1 {
			return
		}
		itemFunc = tagStr[1:idx]
		tagStr = tagStr[idx:]
	}

	if !(strings.HasPrefix(tagStr, "(") && strings.HasSuffix(tagStr, ")")) {
		return
	}

	options := make(map[string]string)
	// @autowire(interface,interface,set=setName)
	// parse tag options
	for _, s := range strings.Split(strings.TrimPrefix(strings.TrimSuffix(tagStr, ")"), "("), ",") {
		if s = strings.TrimSpace(s); len(s) == 0 {
			continue
		}
		spo := strings.Split(s, "=")
		v := ""
		if len(spo) > 1 {
			v = strings.TrimSpace(spo[1])
		}
		options[strings.TrimSpace(spo[0])] = v
	}

	wireElement := element{
		name:    name,
		pkg:     f.Name.Name,
		pkgPath: pkgPath,
	}

	if isFunc {
		wireElement.constructor = name
	} else {
		// found constructor function name with prefix
		for _, constructorPrefix := range []string{"Init", "New"} {
			if ct, ok := f.Scope.Objects[constructorPrefix+name]; ok && ct.Kind == ast.Fun {
				wireElement.constructor = constructorPrefix + name
				break
			}
		}
	}

	// parse set group
	var setName string
	if len(options["set"]) == 0 {
		setName = "unknown"
	} else {
		setName = strcase.LowerCamelCase(options["set"])
	}

	if sc.elementMap[setName] == nil {
		sc.elementMap[setName] = make(map[string]element)
	}

	defer func() {
		log.Printf("wire object collected [ %sSet ] : %s\n", strcase.LowerCamelCase(setName), wireElement.pkg+"."+wireElement.name)
		sc.elementMap[setName][path.Join(pkgPath, name)] = wireElement
	}()

	// parse options
	for key, value := range options {
		switch key {
		case "set":
			continue
		case "new":
			if ct, ok := f.Scope.Objects[value]; ok && ct.Kind == ast.Fun {
				wireElement.constructor = value
			}
			continue
		default:
			wireElement.implements = append(wireElement.implements, key)
		}
	}

	// parse item func
	// @autowire.init as InitEntry
	// @autowire.config as InitEntryConfigParams
	switch itemFunc {
	case "init":
		wireElement.initWire = true
	case "config":
		if decl.typeSpec == nil {
			break
		}
		st, isStruct := decl.typeSpec.Type.(*ast.StructType)
		if !isStruct || st.Fields == nil || len(st.Fields.List) == 0 {
			break
		}
		wireElement.configWire = true
		for _, f := range st.Fields.List {
			fieldName := fmt.Sprintf("%s", f.Type)
			if f.Names != nil {
				fieldName = f.Names[0].String()
			}
			if fieldName[0] >= 'A' && fieldName[0] <= 'Z' {
				wireElement.fields = append(wireElement.fields, fieldName)
			}
		}
	}

	if len(implementMap[name]) > 0 {
		insertIfUnExist(implementMap[name], &wireElement.implements)
	}
}

// analyse implement assign in same file like: var _ io.Writer = &myWriter{}
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
