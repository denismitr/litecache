package litecache

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidConfig = errors.New("invalid config")
)

type Config[T any] struct {
	shards  int
	onEvict func(key string, value T)
}

func NewDefaultConfig[T any]() Config[T] {
	return Config[T]{
		shards: 50,
	}
}

func (c Config[T]) WithShards(shards int) Config[T] {
	c.shards = shards
	return c
}

func (c Config[T]) WithOnEvict(f func(key string, value T)) Config[T] {
	c.onEvict = f
	return c
}

func (c Config[T]) validate() error {
	if c.shards < 1 {
		return fmt.Errorf("%w: shards should be greater or equal to 1", ErrInvalidConfig)
	}

	return nil
}
