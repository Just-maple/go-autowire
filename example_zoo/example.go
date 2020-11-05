package example_zoo

type (
	// it will be collect into zooSet
	// use init to create initZoo method in wire.gen.go
	// @autowire.init(set=zoo)
	Zoo struct {
		Cat       Cat
		Dog       Dog
		Lion      Lion
		FlyAnimal FlyAnimal
	}

	// @autowire.init(set=zoo)
	MiniZoo struct {
		Cat       Cat
		FlyAnimal FlyAnimal
	}

	// it will be collect into animalsSet
	// @autowire(set=animals)
	Cat struct {
	}

	// @autowire(set=animals,FlyAnimal)
	Bird struct {
	}

	FlyAnimal interface {
		Fly()
	}

	// use provider func
	Dog struct{}
)

// it will be collect into animalsSet
// user provider func
// @autowire(set=animals)
func ProvideDog() Dog {
	return Dog{}
}

// it will be collect into animalsSet
// as it has a New method it will use NewLion as provider
// @autowire(set=animals)
type Lion struct{}

func NewLion() Lion {
	return Lion{}
}

// it will be collect into animalsSet and wire as interface FlyAnimal
func (b Bird) Fly() {}
