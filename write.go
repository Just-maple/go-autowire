package gutowire

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/imports"
)

func (sc *searcher) clean() (err error) {
	dirs, err := ioutil.ReadDir(sc.genPath)
	if err != nil {
		return
	}
	if len(dirs) == 0 {
		return
	}
	_ = os.Remove("wire_gen.go")
	for _, f := range dirs {
		if strings.Contains(f.Name(), filePrefix+"_") && strings.Contains(f.Name(), ".go") {
			_ = os.Remove(filepath.Join(sc.genPath, f.Name()))
		}
	}
	return
}

func (sc *searcher) write() (err error) {
	log.Printf("please wait for file [ %s ] writing ...", sc.genPath)
	sc.sets = nil
	_ = os.MkdirAll(sc.genPath, 0775)
	_ = sc.clean()
	for set, m := range sc.elementMap {
		err = sc.writeSet(set, m)
		if err != nil {
			return
		}
	}
	return sc.writeSets()
}

func (sc *searcher) writeSets() (err error) {
	if len(sc.sets) == 0 {
		return
	}
	sort.Strings(sc.sets)
	var (
		fileName   = filepath.Join(sc.genPath, filePrefix+"_sets.go")
		wgFileName = filepath.Join(sc.genPath, "wire.gen.go")
		data       = wireSet{
			Package: sc.pkg,
			SetName: "Sets",
			Items:   []template.HTML{template.HTML(strings.Join(sc.sets, ",\n\t"))},
		}
		bf = bytes.NewBuffer(nil)
	)
	err = setTemp.Execute(bf, &data)
	if err != nil {
		return
	}
	src, err := imports.Process("", bf.Bytes(), nil)
	if err != nil {
		log.Printf("write set error:\n%s", bf.String())
		return err
	}
	err = ioutil.WriteFile(fileName, src, 0664)
	if err != nil || len(sc.initElements) == 0 || !sc.initWire {
		return
	}
	inits := []string{fmt.Sprintf(initTemplateHead, sc.pkg)}
	for _, w := range sc.initElements {
		inits = append(inits, fmt.Sprintf(initItemTemplate, w.name, appendPkg(w.pkg, w.name)))
	}
	wireGenData := strings.Join(inits, "\n")
	src, err = imports.Process("", []byte(wireGenData), nil)
	if err != nil {
		log.Printf("write set error:\n%s", wireGenData)
		return err
	}
	err = ioutil.WriteFile(wgFileName, src, 0664)
	return
}

func (sc *searcher) writeSet(set string, m map[string]element) (err error) {
	var (
		order = make([]string, 0, len(m))

		pkgMap = make(map[string]map[string]string)

		setName  = strings.Title(strcase.ToCamel(set)) + "Set"
		fileName = filepath.Join(sc.genPath, filePrefix+"_"+strcase.ToSnake(set)+".go")
		fs       = token.NewFileSet()
	)
	log.Printf("generating [ %s ]", fileName)
	for key := range m {
		order = append(order, key)
	}
	sort.Strings(order)
	// fix import name
	// support duplicate package name as
	// import (
	// 		pkg  "xxx/pkg"
	//		pkg2 "xxx/xxx/pkg"
	// 		pkg3 "xxx/xxx/xxx/pkg
	// )
	for _, key := range order {
		t := m[key]
		pkg, ok := pkgMap[t.pkg][t.pkgPath]
		if len(pkgMap[t.pkg]) == 0 {
			pkg = t.pkg
			pkgMap[t.pkg] = map[string]string{
				t.pkgPath: t.pkg,
			}
			ok = true
		}
		if ok {
			t.pkg = pkg
			m[key] = t
			continue
		}
		c := len(pkgMap[t.pkg]) + 1
		newPkg := t.pkg + strconv.Itoa(c)
		pkgMap[t.pkg][t.pkgPath] = newPkg
		t.pkg = newPkg
		m[key] = t
	}
	var (
		importPkgs []*ast.ImportSpec
		src        = bytes.NewBuffer(nil)
		data       = wireSet{
			Package: sc.pkg,
			SetName: setName,
		}
	)
	pathPkg := sc.getPkgPath(fileName)
	for _, key := range order {
		// todo:support struct fields
		// generate wire define
		elem := m[key]
		var wireItem []string
		if elem.pkgPath == pathPkg {
			elem.pkg = ""
		}
		stName := appendPkg(elem.pkg, elem.name)
		if elem.constructor != "" {
			wireItem = append(wireItem, appendPkg(elem.pkg, elem.constructor))
		} else {
			wireItem = append(wireItem, fmt.Sprintf(`wire.Struct(new(%s), "*")`, stName))
		}
		for _, itf := range elem.implements {
			var itfName string
			if strings.Contains(itf, ".") {
				itfName = itf
			} else {
				itfName = appendPkg(elem.pkg, itf)
			}
			wireItem = append(wireItem, fmt.Sprintf(`wire.Bind(new(%s), new(*%s))`, itfName, stName))
		}
		data.Items = append(data.Items, template.HTML(strings.Join(wireItem, ",\n\t")))

		if elem.initWire {
			sc.initElements = append(sc.initElements, elem)
		}

		if len(elem.pkg) == 0 {
			continue
		}
		// add import to set file
		imp := &ast.ImportSpec{
			Path: &ast.BasicLit{
				Kind:  token.STRING,
				Value: fmt.Sprintf(`"%s"`, elem.pkgPath),
			},
		}
		_, last := filepath.Split(elem.pkgPath)
		if last != elem.pkg {
			imp.Name = ast.NewIdent(elem.pkg)
		}
		importPkgs = append(importPkgs, imp)
	}

	sc.sets = append(sc.sets, setName)
	err = setTemp.Execute(src, data)
	if err != nil {
		return
	}

	// fill the imports from file searching
	f, err := parser.ParseFile(fs, "", src, parser.ParseComments)
	if err != nil {
		return
	}
	if decl, ok := f.Decls[0].(*ast.GenDecl); ok {
		for _, imp := range importPkgs {
			decl.Specs = append(decl.Specs, imp)
		}
	}
	var bf bytes.Buffer
	err = format.Node(&bf, fs, f)
	if err != nil {
		return
	}
	// finished imports
	ret, err := imports.Process("", bf.Bytes(), nil)
	if err != nil {
		log.Printf("write set error:\n%s", bf.String())
		return
	}
	err = ioutil.WriteFile(fileName, ret, 0664)
	if err != nil {
		return
	}
	return
}

func appendPkg(pkg string, sel string) string {
	if len(pkg) == 0 {
		return sel
	}
	return pkg + "." + sel
}
