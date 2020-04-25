package test_b

import (
	"github.com/Just-maple/go-autowire/test/test_b/test"
	"github.com/Just-maple/go-autowire/test/test_b/test/test_b"
)

// @autowire(set=struct)
type Test struct {
	Test2
	test.Test
}

// @autowire(set=struct)
type Test2 struct {
	test_b.Test
}
