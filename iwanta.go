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

type iwantA struct {
	wantInputIdent string
	wantName       string
	typeName       string
	callFileLines  []string
	callLine       int
	callFile       string
}

func (iw *iwantA) initWantArgIdent() {
	callLineStr := regexpCall.FindAllStringSubmatch(strings.TrimSpace(iw.callFileLines[iw.callLine-1]), -1)
	for i := range callLineStr {
		if len(callLineStr[i]) == 2 {
			iw.wantInputIdent = callLineStr[i][1]
			break
		}
	}

	// rewrite caller replace IWantA with thisIsYour
	if iw.wantInputIdent == "" {
		iw.wantInputIdent = "nil"
	} else {
		iw.wantInputIdent = "&" + iw.wantInputIdent
	}

}

func IWantA(in interface{}, searchDepDirs ...string) (_ struct{}) {
	if len(searchDepDirs) == 0 {
		modPath := getGoModDir()
		if len(modPath) > 0 {
			searchDepDirs = append(searchDepDirs, modPath)
		}
	}

	_, callFile, callLine, ok := runtime.Caller(1)
	if !ok {
		panic("error call path")
	}

	var (
		callFileData, _  = ioutil.ReadFile(callFile)
		fileSet          = token.NewFileSet()
		astCallFile, err = parser.ParseFile(fileSet, "", callFileData, parser.ParseComments)

		iw = &iwantA{
			callFile:      callFile,
			callLine:      callLine,
			callFileLines: strings.Split(string(callFileData), "\n"),
		}
	)

	iw.initWantArgIdent()

	if err != nil {
		panic(err)
	}

	// gen wire.go
	var (
		wantTypeVar string

		genPackage    = astCallFile.Name.Name
		rType         = reflect.TypeOf(in).Elem()
		modeBase, _   = getModBase()
		iwantaPkgPath = getPkgPath(callFile, modeBase)
	)

	if rType.PkgPath() == iwantaPkgPath {
		wantTypeVar = rType.Name()
	} else {
		wantTypeVar = rType.String()
	}

	var (
		wantTypeName = strcase.ToSnake(strings.Replace(strings.Replace(wantTypeVar, "_", "", -1), ".", "_", -1))
		genPath      = filepath.Dir(callFile)
		wireOpt      = []Option{WithPkg(genPackage)}
	)

	iw.typeName = strcase.ToCamel(wantTypeName)

	// clean tmp
	defer func() {
		iw.cleanIWantATemp(callFile)
		if err := recover(); err != nil {
			panic(err)
		} else {
			os.Exit(0)
		}
	}()

	initTmpFileName := filepath.Join(genPath, "wire_init_tmp.go")
	if err = importAndWrite(initTmpFileName, []byte(fmt.Sprintf(initTemplate, genPackage, iw.typeName, wantTypeVar))); err != nil {
		panic(err)
	}

	for _, s := range searchDepDirs {
		wireOpt = append(wireOpt, WithSearchPath(s))
	}

	// run autowire
	if err = RunWire(genPath, wireOpt...); err != nil {
		panic(err)
	}

	// gen init
	if err = iw.writeInitFile(wantTypeVar, wantTypeName); err != nil {
		panic(err)
	}

	if err = iw.updateCallFile(); err != nil {
		panic(err)
	}

	return struct{}{}
}

func (iw *iwantA) updateCallFile() (err error) {
	callLine := strings.TrimSpace(iw.callFileLines[iw.callLine-1])
	assignStr := fmt.Sprintf("_, _ = thisIsYour%s(%s)", iw.typeName, iw.wantInputIdent)

	if strings.HasPrefix(callLine, "var ") {
		assignStr = "var " + assignStr
	}

	iw.callFileLines[iw.callLine-1] = "// " + callLine
	iw.callFileLines = append(iw.callFileLines[:iw.callLine], append([]string{assignStr}, iw.callFileLines[iw.callLine:]...)...)
	return importAndWrite(iw.callFile, []byte(strings.Join(iw.callFileLines, "\n")))
}

func (iw *iwantA) writeInitFile(wantVar, name string) (err error) {
	genPath := filepath.Dir(iw.callFile)
	initFileData, err := ioutil.ReadFile(filepath.Join(genPath, "wire_gen.go"))
	isTest := strings.HasSuffix(iw.callFile, "_test.go")
	if err != nil {
		return
	}
	filename := fmt.Sprintf("%s_init", strcase.ToSnake(name))
	if isTest {
		filename += "_test"
	}
	filename += ".go"
	initFileData = append(initFileData, fmt.Sprintf(thisIsYourTemplate, iw.typeName, wantVar, iw.typeName)...)
	initFileName := filepath.Join(genPath, filename)
	if err = importAndWrite(initFileName, initFileData); err != nil {
		return
	}
	return
}

func (iw *iwantA) cleanIWantATemp(f string) {
	dir := filepath.Dir(f)
	infos, _ := ioutil.ReadDir(dir)
	for _, info := range infos {
		if strings.HasPrefix(info.Name(), "autowire") || info.Name() == "wire_gen.go" || info.Name() == "wire_init_tmp.go" {
			_ = os.Remove(filepath.Join(dir, info.Name()))
		}
	}
}
