package gutowire

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/iancoleman/strcase"
	"golang.org/x/tools/imports"
)

const (
	initTemplate = `// +build wireinject

package %s

func Initialize%s() (%s, func(), error) {
	panic(wire.Build(Sets))
}
`

	thisIsYourTemplate = `
func thisIsYour%s(res *%s) (err error, cleanup func()) {
	*res, cleanup, err = Initialize%s()
	return
}
`
)

var regexpCall = regexp.MustCompile(`gutowire\.IWantA\(&([a-zA-Z]+).*?\)`)

func IWantA(in interface{}, scope ...string) interface{} {
	if len(scope) == 0 {
		modPath := getGoModDir()
		if len(modPath) > 0 {
			scope = append(scope, modPath)
		}
	}
	_, f, l, ok := runtime.Caller(1)
	if !ok {
		panic("error call path")
	}
	callFileData, _ := ioutil.ReadFile(f)
	spln := strings.Split(string(callFileData), "\n")
	found := regexpCall.FindAllStringSubmatch(strings.TrimSpace(spln[l-1]), -1)
	var input string
	for i := range found {
		if len(found[i]) == 2 {
			input = found[i][1]
			break
		}
	}
	fset := token.NewFileSet()
	astCallFile, err := parser.ParseFile(fset, "", callFileData, parser.ParseComments)
	if err != nil {
		panic(err)
	}
	// gen wire.go
	var wantVar string
	genPackage := astCallFile.Name.Name
	rType := reflect.TypeOf(in).Elem()
	base, _ := getModBase()
	gopkg := getPkgPath(f, base)
	if rType.PkgPath() == gopkg {
		wantVar = rType.Name()
	} else {
		wantVar = rType.String()
	}
	spl := strings.Split(wantVar, ".")
	name := spl[len(spl)-1]
	genPath := filepath.Dir(f)

	// clean tmp
	defer func() {
		cleanIWantATemp(f)
		err := recover()
		if err != nil {
			panic(err)
		} else {
			os.Exit(0)
		}
	}()

	src := []byte(fmt.Sprintf(initTemplate, genPackage, name, wantVar))
	res, err := imports.Process("", src, nil)
	if err != nil {
		fmt.Printf("%s",src)
		panic(err)
	}
	_ = ioutil.WriteFile(filepath.Join(genPath, "wire_init_tmp.go"), res, 0664)

	// run autowire
	RunWire(genPath, WithSearchPath(scope[0]), WithPkg(genPackage))

	// gen init
	wiregenData, _ := ioutil.ReadFile(filepath.Join(genPath, "wire_gen.go"))
	wiregenData = append(wiregenData, fmt.Sprintf(thisIsYourTemplate, name, wantVar, name)...)
	genfile := filepath.Join(filepath.Dir(f), fmt.Sprintf("init_%s_test.go", strcase.ToSnake(name)))
	wiregenData, err = imports.Process("", wiregenData, nil)
	if err != nil {
		fmt.Printf("%s",src)
		panic(err)
	}
	err = ioutil.WriteFile(genfile, wiregenData, 0664)
	if err != nil {
		panic(err)
	}

	// rewrite caller replace IWantA with thisIsYour
	if input == "" {
		input = "nil"
	} else {
		input = "&" + input
	}
	spln[l-1] = "// " + strings.TrimSpace(spln[l-1])
	d := fmt.Sprintf("var _, _ = thisIsYour%s(%s)", name, input)
	spln = append(spln[:l], append([]string{d}, spln[l:]...)...)
	res, err = imports.Process("", []byte(strings.Join(spln, "\n")), nil)
	if err != nil {
		fmt.Printf("%s",src)
		panic(err)
	}
	err = ioutil.WriteFile(f, res, 0664)
	if err != nil {
		panic(err)
	}
	return nil
}

func cleanIWantATemp(f string) {
	dir := filepath.Dir(f)
	infos, _ := ioutil.ReadDir(dir)
	for _, info := range infos {
		if strings.HasPrefix(info.Name(), "autowire") || info.Name() == "wire_gen.go" || info.Name() == "wire_init_tmp.go" {
			_ = os.Remove(filepath.Join(dir, info.Name()))
		}
	}
}
