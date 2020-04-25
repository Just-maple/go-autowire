//+build wireinject

package autowire_gen

import (
	"github.com/Just-maple/go-autowire/example/dependencies"
	"github.com/google/wire"
)

func InitTest() dependencies.Test {
	panic(wire.Build(Sets))
}
