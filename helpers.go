package gutowire

import (
	"bytes"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
)

var modTmp string

func getModBase() (modBase string, err error) {
	modpath := getGoModFilePath()
	mb, _ := ioutil.ReadFile(modpath)
	f, err := modfile.Parse("", mb, func(path, version string) (s string, e error) {
		return version, nil
	})
	if err != nil {
		return
	}
	if f.Module == nil {
		err = errors.New("parse mod error,please check your go env")
		return
	}
	modBase = f.Module.Mod.Path
	return
}

func getGoModDir() (modPath string) {
	mod := getGoModFilePath()
	modPath = filepath.Dir(mod)
	return
}

func getGoModFilePath() (modPath string) {
	if len(modTmp) > 0 {
		return modTmp
	}
	cmd := exec.Command("go", "env", "GOMOD")
	stdout := &bytes.Buffer{}
	cmd.Stdout = stdout
	_ = cmd.Run()
	mod := stdout.String()
	mod = strings.Trim(mod, "\n")
	modTmp = mod
	return mod
}

func getPathGoPkgName(pathStr string) (pkg string, err error) {
	info, err := ioutil.ReadDir(pathStr)
	// todo:if not exist return getGoPkgNameByDir
	if err != nil {
		return
	}
	if len(info) == 0 {
		return getGoPkgNameByDir(pathStr), nil
	}
	for _, f := range info {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".go") {
			bs, err := ioutil.ReadFile(filepath.Join(pathStr, f.Name()))
			if err != nil {
				return "", err
			}
			f, err := parser.ParseFile(token.NewFileSet(), "", bs, parser.ParseComments)
			if err != nil {
				return "", err
			}
			return f.Name.Name, nil
		}
	}
	return
}

func getPkgPath(filePath, modBase string) (pkgPath string) {
	abs, err := filepath.Abs(filePath)
	if err != nil {
		return
	}
	dir := getGoModDir()
	if len(abs) < len(dir) {
		return
	}
	pkgPath = filepath.ToSlash(filepath.Dir(filepath.Join(modBase, abs[len(dir):])))
	return
}

func getGoPkgNameByDir(pathStr string) (pkg string) {
	return filepath.Base(pathStr)
}

func importAndWrite(filename string, src []byte) (err error) {
	var writeData []byte
	if writeData, err = imports.Process("", src, nil); err != nil {
		fmt.Printf("%s", src)
		return
	}

	err = ioutil.WriteFile(filename, writeData, os.FileMode(0664))
	return
}
