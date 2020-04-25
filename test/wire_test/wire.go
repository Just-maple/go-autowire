//+build wireinject

package wire_test

import (
	"github.com/Just-maple/go-autowire/test"
	"github.com/google/wire"
)

func InitTest() test.Test {
	panic(wire.Build(Sets))
}
