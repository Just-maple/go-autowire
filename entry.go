package gutowire

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	log.SetPrefix("[gutowire] ")
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}

func newGenOpt(genPath string, opts ...Option) *opt {
	o := &opt{genPath: genPath}
	for _, opt := range opts {
		opt(o)
	}
	o.init()
	return o
}

func (o *opt) init() {
	if len(o.pkg) == 0 {
		var err error
		o.pkg, err = getPathGoPkgName(o.genPath)
		if err != nil {
			o.pkg = strings.ReplaceAll(filepath.Base(o.genPath), "-", "_")
		}
	}
	if len(o.searchPath) == 0 {
		modPath := getGoModDir()
		if len(modPath) > 0 {
			o.searchPath = modPath
		}
	}
}

func RunAutoWire(genPath string, opts ...Option) (err error) {
	if err = RunAutoWireGen(genPath, opts...); err != nil {
		return
	}
	log.Printf("write wire files success")
	return runWire(genPath)
}

func runWire(path string) (err error) {
	log.Printf("start runnning wire")

	p, e := exec.LookPath("wire")
	if e != nil {
		err = fmt.Errorf("wire not found: %v \n%s\n", e,
			"please install wire by [ go get github.com/google/wire/cmd/wire ]")
	}
	cmd := exec.Command(p)
	cmd.Dir = path
	ret, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("[gen failed] %s", ret)
		return
	}
	log.Printf("[gen success] %s", ret)
	return
}
