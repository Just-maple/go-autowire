//+build wireinject

package init_zoo

import (
	"github.com/Just-maple/go-autowire/example_zoo"
	"github.com/google/wire"
)

func InitZoo() example_zoo.Zoo {
	panic(wire.Build(Sets))
}
