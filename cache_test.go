package litecache_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/denismitr/litecache"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCache(t *testing.T) {
	t.Run("single set get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[string](ctx)
		c.Set("foo", "bar")

		assert.Equal(t, 1, c.Len())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, "bar", v)
	})

	t.Run("multi set get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		const iterations = 100_000

		c := litecache.New[string](ctx)
		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Len())

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), fmt.Sprintf("value:%d", i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Len())

		for i := 0; i < iterations; i++ {
			v, found := c.Get(fmt.Sprintf("key:%d", i))
			assert.True(t, found)
			assert.Equal(t, fmt.Sprintf("value:%d", i), v)
		}
	})

	t.Run("setnx single already existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx)
		c.Set("foo", 1)

		assert.Equal(t, 1, c.Len())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 1, v)

		assert.False(t, c.SetNX("foo", 3))
		assert.Equal(t, 1, c.Len())

		//expect value not change
		v2, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 1, v2)
	})
}
