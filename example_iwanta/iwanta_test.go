package example_iwanta

import (
	gutowire "github.com/Just-maple/go-autowire"
	"github.com/Just-maple/go-autowire/example_zoo"
)

var zoo example_zoo.Zoo

// run go test ./example_iwanta/...
// then it will generate the init_zoo_test.go
var _ = gutowire.IWantA(&zoo)
