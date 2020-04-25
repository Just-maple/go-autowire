package test

import (
	"github.com/Just-maple/go-autowire/test/test_b/test/test"
)

// @autowire(set=struct)
type Test struct {
	test.Test
	test.Test2
	T2 Test2
}

// @autowire(set=struct)
type Test2 struct {
	test.Test2
}
