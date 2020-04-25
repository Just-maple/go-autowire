package example_zoo

// it will be collect into zooSet
// @autowire(set=zoo)
type Zoo struct {
	Cat       Cat
	Dog       Dog
	FlyAnimal FlyAnimal
}

// it will be collect into animalsSet
// @autowire(set=animals)
type Cat struct {
}

type FlyAnimal interface {
	Fly()
}

// it will be collect into animalsSet and wire as interface FlyAnimal
// @autowire(set=animals,FlyAnimal)
type Bird struct {
}

func (b Bird) Fly() {
}

// it will be collect into animalsSet
// @autowire(set=animals)
type Dog struct {
}
