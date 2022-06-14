package gutowire

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/stoewer/go-strcase"
)

const (
	thisIsYourTemplate = `
func thisIsYour%s(res *%s,%s) (err error, cleanup func()) {
	*res, cleanup, err = %s
	return
}
`
)

var regexpCall = regexp.MustCompile(`gutowire\.IWantA\(&([a-zA-Z]+).*?\)`)

type iwantA struct {
	wantInputIdent     string
	wantName           string
	thisIsYourFuncName string
	callFileLines      []string
	callLine           int
	callFile           string
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
		callFileData, _ = ioutil.ReadFile(callFile)
		iw              = &iwantA{
			callFile:      callFile,
			callLine:      callLine,
			callFileLines: strings.Split(string(callFileData), "\n"),
		}
	)

	iw.initWantArgIdent()

	// gen wire.go
	var (
		wantTypeVar string
		genSuccess  bool

		rType       = reflect.TypeOf(in).Elem()
		modeBase, _ = getModBase()
		callPkgPath = getPkgPath(callFile, modeBase)
	)

	if rType.PkgPath() == callPkgPath {
		wantTypeVar = rType.Name()
	} else {
		wantTypeVar = rType.String()
	}

	var (
		wantTypeName = strcase.SnakeCase(strings.Replace(strings.Replace(wantTypeVar, "_", "", -1), ".", "_", -1))
		genPath      = filepath.Dir(callFile)
		wireOpt      = make([]Option, 0)
	)

	iw.thisIsYourFuncName = strcase.UpperCamelCase(wantTypeName)

	// clean tmp
	defer func() {
		iw.cleanIWantATemp(callFile)
		if genSuccess {
			os.Exit(0)
		}
	}()

	for _, s := range searchDepDirs {
		wireOpt = append(wireOpt, WithSearchPath(s))
	}

	wireOpt = append(wireOpt, InitStruct(strings.TrimPrefix(wantTypeVar, "*")))

	// run autowire
	if err := RunAutoWire(genPath, wireOpt...); err != nil {
		panic(err)
	}

	// gen init
	args, err := iw.writeInitFile(wantTypeVar, wantTypeName)
	if err != nil {
		panic(err)
	}

	if err = iw.updateCallFile(args); err != nil {
		panic(err)
	}

	genSuccess = true
	return struct{}{}
}

func (iw *iwantA) updateCallFile(configArgs []string) (err error) {
	callLine := strings.TrimSpace(iw.callFileLines[iw.callLine-1])
	callArgs := strings.Join(append([]string{iw.wantInputIdent}, configArgs...), ",")
	assignStr := fmt.Sprintf("_, _ = thisIsYour%s(%s)", iw.thisIsYourFuncName, callArgs)

	if strings.HasPrefix(callLine, "var ") {
		assignStr = "var " + assignStr
	}

	iw.callFileLines[iw.callLine-1] = "// " + callLine
	iw.callFileLines = append(iw.callFileLines[:iw.callLine], append([]string{assignStr}, iw.callFileLines[iw.callLine:]...)...)
	return importAndWrite(iw.callFile, []byte(strings.Join(iw.callFileLines, "\n")))
}

var regexpInitMethod = regexp.MustCompile(`Initialize(.+?)\((.*?)\)`)

func (iw *iwantA) writeInitFile(wantVar, name string) (args []string, err error) {
	genPath := filepath.Dir(iw.callFile)
	initFileData, err := ioutil.ReadFile(filepath.Join(genPath, "wire_gen.go"))
	if err != nil {
		return
	}
	call := ""
	ret := regexpInitMethod.FindStringSubmatch(string(initFileData))
	if len(ret) >= 2 {
		argsVar := make([]string, 0)
		if len(ret) > 2 {
			for _, sp := range strings.Split(ret[2], ",") {
				if spp := strings.SplitN(sp, " ", 2); len(spp) == 2 {
					args = append(args, "&"+strings.TrimPrefix(spp[1], "*")+"{}")
					argsVar = append(argsVar, spp[0])
				}
			}
		}
		call = fmt.Sprintf(`Initialize%s(%s)`, ret[1], strings.Join(argsVar, ","))
	} else {
		err = errors.New("invalid init file")
		return
	}

	filename := fmt.Sprintf("%s_init", strcase.SnakeCase(name))
	if strings.HasSuffix(iw.callFile, "_test.go") {
		filename += "_test"
	}
	filename += ".go"
	initFileData = append(initFileData, fmt.Sprintf(thisIsYourTemplate, iw.thisIsYourFuncName, wantVar, ret[2], call)...)
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
		if strings.HasPrefix(info.Name(), "autowire") ||
			info.Name() == "wire.gen.go" ||
			info.Name() == "wire_gen.go" {
			_ = os.Remove(filepath.Join(dir, info.Name()))
		}
	}
}
