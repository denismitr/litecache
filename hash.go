package litecache

type hasher interface {
	Hash(string) uint64
}
