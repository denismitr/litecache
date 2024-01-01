package litecache

import (
	"sync"
	"time"
)

type item[T any] struct {
	value T
	exp   int64
}

type shard[T any] struct {
	mux   sync.RWMutex
	items map[string]item[T]
}

func newShard[T any]() *shard[T] {
	return &shard[T]{
		items: make(map[string]item[T]),
	}
}

func (s *shard[T]) get(key string) (item[T], bool) {
	s.mux.RLock()
	defer s.mux.RUnlock()
	item, ok := s.items[key]
	if ok && item.exp > 0 && time.Now().UnixNano() > item.exp {
		return item, false
	}

	return item, ok
}

func (s *shard[T]) set(key string, value T, ttl time.Duration) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	var added bool
	if _, exists := s.items[key]; !exists {
		added = true
	}

	exp := int64(-1)
	if ttl > 0 {
		exp = time.Now().UnixNano() + ttl.Nanoseconds()
	}

	s.items[key] = item[T]{value: value, exp: exp}
	return added
}

func (s *shard[T]) setNX(key string, value T, ttl time.Duration) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	// if exists and not expired return false
	if item, exists := s.items[key]; exists {
		if item.exp <= 0 || item.exp > time.Now().UnixNano() {
			return false
		}
	}

	exp := int64(NoExpiration)
	if ttl > 0 {
		exp = time.Now().UnixNano() + ttl.Nanoseconds()
	}

	s.items[key] = item[T]{value: value, exp: exp}
	return true
}

func (s *shard[T]) setEX(key string, value T, ttl time.Duration) bool {
	s.mux.Lock()
	defer s.mux.Unlock()

	itm, exists := s.items[key]
	// if exists and expired return false
	if !exists || (itm.exp > 0 && itm.exp < time.Now().UnixNano()) {
		return false
	}

	exp := int64(NoExpiration)
	if ttl > 0 {
		exp = time.Now().UnixNano() + ttl.Nanoseconds()
	}

	s.items[key] = item[T]{value: value, exp: exp}
	return true
}

func (s *shard[T]) cleanExpired() int {
	s.mux.Lock()
	defer s.mux.Unlock()
	deleted := 0
	now := time.Now().UnixNano()
	for k, item := range s.items {
		if item.exp > 0 && item.exp < now {
			delete(s.items, k)
			deleted++
		}
	}
	return deleted
}
