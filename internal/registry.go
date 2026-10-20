package internal

var registry []Migration
var seedRegistry []Seed

func Register(m Migration) {
	registry = append(registry, m)
}

func RegisterSeed(s Seed) {
	seedRegistry = append(seedRegistry, s)
}

func All() []Migration {
	return registry
}

func AllSeeds() []Seed {
	return seedRegistry
}

func Reset() {
	registry = nil
	seedRegistry = nil
}
