package litecache

import (
	"context"
	"sync/atomic"
	"time"
)

const (
	NoExpiration time.Duration = -1
)

type Cache[T any] struct {
	hasher    hasher
	shards    []*shard[T]
	shardMask uint64
	len       atomic.Int64
}

func New[T any](ctx context.Context) *Cache[T] {
	c := &Cache[T]{
		shardMask: uint64(50 - 1),
		shards:    make([]*shard[T], 50),
		hasher:    newDefaultHasher(),
	}

	j := newJanitor[T](ctx, 1*time.Second)
	for i := range c.shards {
		c.shards[i] = newShard[T]()
		j.runOn(c.shards[i], func(evicted int) {
			c.len.Add(-int64(evicted))
		})
	}

	return c
}

func (c *Cache[T]) getShard(key string) *shard[T] {
	hk := c.hasher.Hash(key)
	return c.shards[hk&c.shardMask]
}

func (c *Cache[T]) Get(key string) (T, bool) {
	shard := c.getShard(key)
	item, found := shard.get(key)
	if !found {
		return zeroV[T](), false
	}

	return item.value, true
}

func (c *Cache[T]) Set(key string, value T) {
	shard := c.getShard(key)
	if shard.set(key, value, NoExpiration) {
		c.len.Add(1)
	}
}

func (c *Cache[T]) SetNX(key string, value T) bool {
	shard := c.getShard(key)
	if shard.setNX(key, value, NoExpiration) {
		c.len.Add(1)
		return true
	}

	return false
}

func (c *Cache[T]) SetTTL(key string, value T, ttl time.Duration) {
	shard := c.getShard(key)
	if shard.set(key, value, ttl) {
		c.len.Add(1)
	}
}

func (c *Cache[T]) Len() int {
	return int(c.len.Load())
}

func zeroV[T any]() T {
	var v T
	return v
}
