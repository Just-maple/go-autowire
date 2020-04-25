package gutowire

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func init() {
	log.SetPrefix("[gutowire] ")
	log.SetFlags(0)
	log.SetOutput(os.Stdout)
}

var searcherStore = make(map[string]*searcher)

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
		o.pkg, err = filepath.Abs(o.genPath)
		if err != nil {
			o.pkg = filePrefix
		} else {
			o.pkg = filepath.Base(o.pkg)
		}
	}
	if len(o.searchPath) == 0 {
		modPath := GetGoModDir()
		if len(modPath) > 0 {
			o.searchPath = modPath
		}
	}
}

func RunWire(genPath string, opts ...Option) {
	o := newGenOpt(genPath, opts...)
	file := o.searchPath
	pkg := o.pkg
	err := SearchAllPath(file, genPath, pkg, opts...)
	if err != nil {
		panic(err)
	}
	log.Printf("write wire files success")
	log.Printf("start runnning wire")
	p, e := exec.LookPath("wire")
	if e != nil {
		panic(fmt.Errorf("wire not found: %v \n%s\n", e,
			"please install wire by [ go get github.com/google/wire/cmd/wire ]"))
	}
	cmd := exec.Command(p)
	var s bytes.Buffer
	cmd.Dir = genPath
	cmd.Stderr = &s
	err = cmd.Run()
	if err != nil {
		log.Printf("[gen failed] %s", s.String())
		panic(err)
	}
	log.Printf("[gen success] %s", s.String())
}
