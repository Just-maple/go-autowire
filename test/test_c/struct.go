package test_c

import "github.com/Just-maple/go-autowire/test/test_b"

// @autowire(set=struct)
type Test struct {
	Test2
}

// @autowire(set=struct)
type Test2 struct {
	T3 test_b.Test2
}
