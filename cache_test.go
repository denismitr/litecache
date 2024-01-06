package litecache_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/denismitr/litecache"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCache(t *testing.T) {
	t.Run("single set get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[string](ctx, 50)
		c.Set("foo", "bar")

		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, "bar", v)
	})

	t.Run("multi set get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		const iterations = 100_000

		c := litecache.New[string](ctx, 50)
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Count())

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Count())

		for i := 0; i < iterations; i++ {
			v, found := c.Get(fmt.Sprintf("key:%d", i))
			assert.True(t, found)
			assert.Equal(t, fmt.Sprintf("value:%d", i), v)
		}
	})

	t.Run("setnx single already existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.Set("foo", 1)

		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 1, v)

		assert.False(t, c.SetNx("foo", 3))
		assert.Equal(t, 1, c.Count())

		//expect value not change
		v2, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 1, v2)
	})

	t.Run("setex single already existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.Set("foo", 1)

		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 1, v)

		assert.True(t, c.SetEx("foo", 3))
		assert.Equal(t, 1, c.Count())

		//expect value not change
		v2, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 3, v2)
	})

	t.Run("setex single non existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.Set("foo", 1)

		assert.Equal(t, 1, c.Count())

		vFoo, foundFoo := c.Get("foo")
		assert.True(t, foundFoo)
		assert.Equal(t, 1, vFoo)

		assert.False(t, c.SetEx("bar", 3))
		assert.Equal(t, 1, c.Count())

		//expect value not change
		vFoo2, foundFoo := c.Get("foo")
		assert.True(t, foundFoo)
		assert.Equal(t, 1, vFoo2)

		vBar, foundBar := c.Get("bar")
		assert.False(t, foundBar)
		assert.Equal(t, 0, vBar)
	})

	t.Run("set ttl and get single key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.SetTtl("foo", 10, 2*time.Second)
		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 10, v)

		time.Sleep(1 * time.Second)

		v2, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 10, v2)

		time.Sleep(1 * time.Second)

		v3, found := c.Get("foo")
		assert.False(t, found)
		assert.Equal(t, 0, v3)
	})
}

func TestCache_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("many keys sync", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const N = 1_000_000

		c := litecache.New[int](ctx, 75)
		for i := 1; i <= N; i++ {
			c.SetNx(fmt.Sprintf("key:%d", i), i)
		}

		assert.Equal(t, N, c.Count())

		totalFound := 0
		res := make(map[string]int)
		c.ForEach(func(k string, v int) {
			totalFound++
			res[k] = v
		})
		assert.Equal(t, N, totalFound)

		for i := 1; i <= N; i++ {
			v, ok := res[fmt.Sprintf("key:%d", i)]
			assert.True(t, ok)
			assert.Equal(t, i, v)
		}
	})

	t.Run("all expired keys except 2", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const N = 1_000_000

		c := litecache.New[int](ctx, 75)
		for i := 1; i <= N; i++ {
			c.SetNxTtl(fmt.Sprintf("key:%d", i), i, 20*time.Millisecond)
		}

		time.Sleep(21 * time.Millisecond)

		c.SetNx("key:foo", 100)
		c.SetNx("key:bar", 200)

		totalFound := 0
		res := make(map[string]int)
		c.ForEach(func(k string, v int) {
			totalFound++
			res[k] = v
		})
		assert.Equal(t, 2, totalFound)

		for i := 1; i <= N; i++ {
			v, ok := res[fmt.Sprintf("key:%d", i)]
			assert.False(t, ok)
			assert.Equal(t, 0, v)
		}

		{
			v, found := c.Get("key:foo")
			assert.True(t, found)
			assert.Equal(t, 100, v)
		}

		{
			v, found := c.Get("key:bar")
			assert.True(t, found)
			assert.Equal(t, 200, v)
		}
	})
}

func TestCache_Transform(t *testing.T) {
	t.Run("transform single value", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.SetTtl("foo", 3, 2*time.Second)
		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 3, v)

		assert.True(t, c.Transform("foo", func(n int) int {
			n += 20
			return n
		}))

		v2, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 23, v2)
	})

	t.Run("transform expired value should fail", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx, 50)
		c.SetTtl("foo", 3, 10*time.Millisecond)
		assert.Equal(t, 1, c.Count())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 3, v)

		time.Sleep(30 * time.Millisecond)

		// already expired
		assert.False(t, c.Transform("foo", func(n int) int {
			n += 20
			return n
		}))

		// already expired
		v2, found := c.Get("foo")
		assert.False(t, found)
		assert.Equal(t, 0, v2)
	})
}
