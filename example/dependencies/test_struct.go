package dependencies

import (
	"github.com/Just-maple/go-autowire/example/dependencies/test_b"
	"github.com/Just-maple/go-autowire/example/dependencies/test_b/test"
	"github.com/Just-maple/go-autowire/example/dependencies/test_c"
)

// @autowire.init(set=struct)
type Test struct {
	T4 test.Test
	Test2
	Test3
	Test4
	test_c.Test
	T1 test_b.Test
	T3 test_b.Test2
	T2 test_c.Test2
	T  TestInterface1
}

type TestInterface1 interface {
}

// @autowire(set=struct,TestInterface1)
type Test2 struct{ Test3 }

// @autowire(set=struct,new=ConstTest3)
type Test3 struct{}

func ConstTest3() Test3 {
	return Test3{}
}

// @autowire(set=struct)
type Test4 struct{}

func NewTest4() Test4 {
	return Test4{}
}

//@autowire(set=func)
func UselessFunc() interface{} {
	return nil
}
