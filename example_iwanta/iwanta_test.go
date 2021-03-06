package example_iwanta

import (
	gutowire "github.com/Just-maple/go-autowire"
	"github.com/Just-maple/go-autowire/example/dependencies"
	"github.com/Just-maple/go-autowire/example_zoo"
)

var zoo example_zoo.Zoo
var testI dependencies.Test
var local Local

// run go test ./example_iwanta/...
// then it will generate the init file
var _ = gutowire.IWantA(&zoo)
var _ = gutowire.IWantA(&local)
var _ = gutowire.IWantA(&testI)
