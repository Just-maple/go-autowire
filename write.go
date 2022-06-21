package gutowire

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/stoewer/go-strcase"
)

func (sc *autoWireSearcher) clean() (err error) {
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

func (sc *autoWireSearcher) write() (err error) {
	log.Printf("please wait for file [ %s ] writing ...", sc.genPath)
	sc.sets = nil
	_ = os.MkdirAll(sc.genPath, 0775)
	_ = sc.clean()

	for set, m := range sc.elementMap {
		set := set
		m := m
		sc.wg.Go(func() error {
			return sc.writeSet(set, m)
		})
	}

	if err = sc.wg.Wait(); err != nil {
		return
	}

	return sc.writeSets()
}

func (sc *autoWireSearcher) writeSets() (err error) {
	if len(sc.sets) == 0 {
		return
	}

	sc.wg.Go(func() (err error) {
		sort.Strings(sc.sets)

		var (
			fileName = filepath.Join(sc.genPath, filePrefix+"_sets.go")
			bf       = bytes.NewBuffer(nil)

			set = wireSet{
				Package: sc.pkg,
				SetName: "Sets",
				Items:   []string{strings.Join(sc.sets, ",\n\t")},
			}
		)

		if err = setTemp.Execute(bf, &set); err != nil {
			return
		}

		if err = importAndWrite(fileName, bf.Bytes()); err != nil {
			return
		}
		return
	})

	sc.wg.Go(func() (err error) {
		if len(sc.initElements) == 0 || len(sc.initWire) == 0 {
			return
		}

		sort.Slice(sc.initElements, func(i, j int) bool {
			return sc.initElements[i].name < sc.initElements[j].name
		})

		inits := []string{fmt.Sprintf(initTemplateHead, sc.pkg)}
		configs := make([]string, 0, len(sc.configElements))

		sort.Slice(sc.configElements, func(i, j int) bool {
			return sc.configElements[i].name < sc.configElements[j].name
		})

		for i, c := range sc.configElements {
			configs = append(configs, fmt.Sprintf(`c%d *%s`, i, appendPkg(c.pkg, c.name)))
		}

		paramConfig := strings.Join(configs, ",")

		if len(sc.initWire) == 1 && sc.initWire[0] == "*" {
			for _, w := range sc.initElements {
				inits = append(inits, fmt.Sprintf(initItemTemplate, w.name, paramConfig, "*"+appendPkg(w.pkg, w.name)))
			}
		} else {
			for _, i := range sc.initWire {
				sp := strings.Split(i, ".")
				inits = append(inits, fmt.Sprintf(initItemTemplate, sp[len(sp)-1], paramConfig, i))
			}
		}

		wireGenData := strings.Join(inits, "\n")
		err = importAndWrite(filepath.Join(sc.genPath, "wire.gen.go"), []byte(wireGenData))
		return
	})

	return sc.wg.Wait()
}

func (sc *autoWireSearcher) writeSet(set string, elements map[string]element) (err error) {
	var (
		order  = make([]string, 0, len(elements))
		pkgMap = make(map[string]map[string]string)

		setName  = strings.Title(strcase.UpperCamelCase(set)) + "Set"
		fileName = filepath.Join(sc.genPath, filePrefix+"_"+strcase.SnakeCase(set)+".go")
		fs       = token.NewFileSet()
	)

	log.Printf("generating %s [ %s ]", setName, fileName)

	for key := range elements {
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
	for _, elementKey := range order {
		elem := elements[elementKey]
		pkg, ok := pkgMap[elem.pkg][elem.pkgPath]
		if len(pkgMap[elem.pkg]) == 0 {
			pkg = elem.pkg
			pkgMap[elem.pkg] = map[string]string{
				elem.pkgPath: elem.pkg,
			}
			ok = true
		}
		if ok {
			elem.pkg = pkg
			elements[elementKey] = elem
			continue
		}
		fixPkgDuplicate := len(pkgMap[elem.pkg]) + 1
		newPkg := elem.pkg + strconv.Itoa(fixPkgDuplicate)
		pkgMap[elem.pkg][elem.pkgPath] = newPkg
		elem.pkg = newPkg
		elements[elementKey] = elem
	}

	var (
		importPkgs []*ast.ImportSpec

		src     = bytes.NewBuffer(nil)
		pathPkg = sc.getPkgPath(fileName)

		data = wireSet{
			Package: sc.pkg,
			SetName: setName,
		}
	)

	for _, key := range order {
		// generate wire define
		var (
			wireItem []string
			elem     = elements[key]
		)

		if elem.pkgPath == pathPkg {
			elem.pkg = ""
		}

		stName := appendPkg(elem.pkg, elem.name)

		if elem.configWire {
			sort.Strings(elem.fields)
			wireItem = append(wireItem, fmt.Sprintf(`wire.FieldsOf(new(*%s),%s)`, stName,
				"\n"+`"`+strings.Join(elem.fields, "\",\n\"")+`",`+"\n"))
			sc.Lock()
			sc.configElements = append(sc.configElements, elem)
			sc.Unlock()
		} else {
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

			if elem.initWire {
				sc.Lock()
				sc.initElements = append(sc.initElements, elem)
				sc.Unlock()
			}
		}

		data.Items = append(data.Items, strings.Join(wireItem, ",\n\t"))

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

	sc.Lock()
	sc.sets = append(sc.sets, setName)
	sc.Unlock()

	if err = setTemp.Execute(src, data); err != nil {
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

	setDataBuf := &bytes.Buffer{}
	if err = format.Node(setDataBuf, fs, f); err != nil {
		return
	}
	// finished imports
	return importAndWrite(fileName, setDataBuf.Bytes())
}

func appendPkg(pkg string, sel string) string {
	if len(pkg) == 0 {
		return sel
	}
	return pkg + "." + sel
}
