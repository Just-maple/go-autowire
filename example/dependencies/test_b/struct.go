package test_b

import (
	test2 "github.com/Just-maple/go-autowire/example/dependencies/test_b/test"
	test_b2 "github.com/Just-maple/go-autowire/example/dependencies/test_b/test/test_b"
)

// @autowire(set=struct)
type Test struct {
	Test2
	test2.Test
}

// @autowire(set=struct)
type Test2 struct {
	test_b2.Test
}
