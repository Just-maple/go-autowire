# Go-AutoWire
> helps you to generate wire files with easy annotate


this project is base on [wire](github.com/google/wire)

but it did `simplify` the wire usage and make wire `much more stronger `

## Installation


```sh
go install github.com/Just-maple/go-autowire/cmd/gutowire
```

## Usage example

If you want to build a `zoo`,you may need some dependencies like animals
```go
package example

type Zoo struct{ 
    Cat         Cat
    Dog         Dog
    FlyAnimal FlyAnimal
}

type Cat struct{
}

type FlyAnimal interface{
    Fly()
}

type Bird struct{
}

func (b Bird)Fly(){
}

type Dog struct{
}
```

in traditional `wire`,you need to write some files to explain the wire relation to google/wire

```go
package example_zoo

import (
	"github.com/google/wire"
)

var zooSet = wire.NewSet(
	wire.Struct(new(Zoo), "*"),
)

var animalsSet = wire.NewSet(
	wire.Struct(new(Cat), "*"),
	wire.Struct(new(Dog), "*"),

	wire.Struct(new(Bird), "*"),
	wire.Bind(new(FlyAnimal), new(Bird)),
)

var sets = wire.NewSet(zooSet, animalsSet)

func InitZoo() Zoo {
	panic(wire.Build(sets))
}
```

you need to rewrite your `wire.go` and comes much more harder to manager all the dependencies

as your zoo goes bigger and bigger 

life seems goes hard

### but now

you can waist your time to continue manage this shit wire sets or 

use `gutowire`

write annotate as `below`
```go
package example

// it will be collect into zooSet (this comment is not necessary)
// @autowire(set=zoo)
type Zoo struct{ 
    Cat         Cat
    Dog         Dog
    FlyAnimal FlyAnimal
}

// it will be collect into animalsSet (this comment is not necessary)
// @autowire(set=animals)
type Cat struct{
}


type FlyAnimal interface{
    Fly()
}

// it will be collect into animalsSet and wire as interface FlyAnimal (this comment is not necessary)
// @autowire(set=animals,FlyAnimal)
type Bird struct{
}

func (b Bird)Fly(){
}

// it will be collect into animalsSet (this comment is not necessary)
// @autowire(set=animals)
type Dog struct{
}
```
and write the only file `example_zoo/init_zoo/wire.go` that you don't edit any more 

```go
//+build wireinject

package init_zoo

import (
	"github.com/Just-maple/go-autowire/example_zoo"
	"github.com/google/wire"
)

func InitZoo() example_zoo.Zoo {
	panic(wire.Build(Sets))
}

```

and run
```sh
gutowire -w ./example_zoo/init_zoo -s ./example_zoo
```

all the wire files you need will genned and use it simply

this's all