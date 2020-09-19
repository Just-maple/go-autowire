package gutowire

import (
	"bytes"
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
	o.fix()
	return o
}

func (o *opt) fix() {
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

func RunWire(genPath string, opts ...Option) (err error) {
	if err = SearchAllPath(genPath, opts...); err != nil {
		return
	}
	log.Printf("write wire files success")
	log.Printf("start runnning wire")
	p, e := exec.LookPath("wire")
	if e != nil {
		err = fmt.Errorf("wire not found: %v \n%s\n", e,
			"please install wire by [ go get github.com/google/wire/cmd/wire ]")
	}
	cmd := exec.Command(p)
	var s bytes.Buffer
	cmd.Dir = genPath
	cmd.Stderr = &s
	err = cmd.Run()
	if err != nil {
		log.Printf("[gen failed] %s", s.String())
		return
	}
	log.Printf("[gen success] %s", s.String())
	return
}
