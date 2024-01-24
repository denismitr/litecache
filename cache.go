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

// New - creates a new cache
func New[T any](ctx context.Context) *Cache[T] {
	cfg := NewDefaultConfig[T]()
	return newWithConfig[T](ctx, cfg)
}

func NewWithConfig[T any](ctx context.Context, cfg Config[T]) (*Cache[T], error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return newWithConfig[T](ctx, cfg), nil
}

func newWithConfig[T any](ctx context.Context, cfg Config[T]) *Cache[T] {
	c := &Cache[T]{
		shardMask: uint64(cfg.shards - 1),
		shards:    make([]*shard[T], cfg.shards),
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

// Get return value for a key, if it exists and has not expired
// zero value is returned if the key was not found or has expired
func (c *Cache[T]) Get(key string) (T, bool) {
	shard := c.getShard(key)
	item, found := shard.get(key)
	if !found {
		return zeroV[T](), false
	}

	return item.value, true
}

func (c *Cache[T]) Transform(key string, effector func(value T) T) bool {
	shard := c.getShard(key)
	return shard.transform(key, effector)
}

func (c *Cache[T]) ForEach(fn func(k string, v T)) {
	for _, s := range c.shards {
		s.iterate(fn)
	}
}

// Set - sets key value pair.
// it will update the value if key already exists in the cache and has not expired.
func (c *Cache[T]) Set(key string, value T) {
	shard := c.getShard(key)
	if shard.set(key, value, NoExpiration) {
		c.len.Add(1)
	}
}

// SetTtl - sets key value pair with ttl.
// it will update the value if key already exists in the cache and has not expired.
func (c *Cache[T]) SetTtl(key string, value T, ttl time.Duration) {
	shard := c.getShard(key)
	if shard.set(key, value, ttl) {
		c.len.Add(1)
	}
}

// SetNx - sets key value pair only if key does not exist in the cache or has expired.
// if the key value pair was set successfully it returns true
func (c *Cache[T]) SetNx(key string, value T) bool {
	shard := c.getShard(key)
	if shard.setNX(key, value, NoExpiration) {
		c.len.Add(1)
		return true
	}

	return false
}

// SetNxTtl - sets key value only if key does not exist in the cache or has expired.
// ttl expected to be given as a last parameter.
// if the key value pair was set successfully it returns true
func (c *Cache[T]) SetNxTtl(key string, value T, ttl time.Duration) bool {
	shard := c.getShard(key)
	if shard.setNX(key, value, ttl) {
		c.len.Add(1)
		return true
	}
	return false
}

// SetEx - updates key value pair if key already exists and not expired in the cache.
// if value was updated, returns true
func (c *Cache[T]) SetEx(key string, value T) bool {
	shard := c.getShard(key)
	return shard.setEX(key, value, NoExpiration)
}

// SetExTtl - updates key value pair if key already exists and not expired in the cache.
// ttl expected to be given as a last parameter.
// if value was updated, returns true
func (c *Cache[T]) SetExTtl(key string, value T, ttl time.Duration) bool {
	shard := c.getShard(key)
	if shard.setEX(key, value, ttl) {
		c.len.Add(1)
		return true
	}
	return false
}

// Remove - removes the value from cache if present
// returns true if key was found and false if it was not
func (c *Cache[T]) Remove(key string) bool {
	s := c.getShard(key)
	_, found := s.remove(key)
	if found {
		c.len.Add(-1)
	}
	return found
}

// GetAndRemove - removes the value from cache if present
// returns value and boolean true if key was found and zero value and boolean false if it was not
func (c *Cache[T]) GetAndRemove(key string) (T, bool) {
	s := c.getShard(key)
	v, found := s.remove(key)
	if found {
		c.len.Add(-1)
	}
	return v, found
}

// Count returns the number of in the cache keys.
// It might get delayed updates when keys expire.
func (c *Cache[T]) Count() int {
	return int(c.len.Load())
}

func zeroV[T any]() T {
	var v T
	return v
}
