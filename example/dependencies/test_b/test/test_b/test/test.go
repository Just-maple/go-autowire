package test

import (
	test2 "github.com/Just-maple/go-autowire/example/dependencies/test_b/test"
)

// @autowire(set=struct)
type Test struct {
	test2.Test
	test2.Test2
	T2 Test2
}

// @autowire(set=struct)
type Test2 struct {
	test2.Test2
}
