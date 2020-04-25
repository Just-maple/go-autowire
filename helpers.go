package gutowire

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"
)

var modTmp string

func GetModBase() (modBase string, err error) {
	modpath := GetGoModFilePath()
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

func GetGoModDir() (modPath string) {
	mod := GetGoModFilePath()
	modPath, _ = filepath.Split(mod)
	return
}

func GetGoModFilePath() (modPath string) {
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
