package test_c

import (
	test_b2 "github.com/Just-maple/go-autowire/example/dependencies/test_b"
)

// @autowire(set=struct)
type Test struct {
	Test2
}

// @autowire(set=struct)
type Test2 struct {
	T3 test_b2.Test2
}
