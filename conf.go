package shardedflight

// // // // // // // //

type KeyBuilderFunc func(parts ...string) string

type HashFunc func(string) uint64

type ConfObj struct {
	BuildKey KeyBuilderFunc // Custom KeyBuilderFunc. If nil, defaultBuilder is used.
	Hash     HashFunc       // Custom HashFunc. If nil, defaultHash is used.

	Shards uint32 // Number of shards (MUST be a power of 2)
}
