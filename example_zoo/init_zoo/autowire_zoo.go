// Code generated by go-autowire. DO NOT EDIT.

package init_zoo

import (
	"github.com/Just-maple/go-autowire/example_zoo"
	"github.com/google/wire"
)

var ZooSet = wire.NewSet(
	wire.Struct(new(example_zoo.Zoo), "*"),
)