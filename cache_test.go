package litecache_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/stretchr/testify/require"
	"sync"
	"testing"
	"time"

	"github.com/denismitr/litecache"
	"github.com/stretchr/testify/assert"
)

func TestDefaultCache(t *testing.T) {
	t.Parallel()

	t.Run("single set get", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[string](ctx)
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

		c := litecache.New[string](ctx)
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

		c := litecache.New[int](ctx)
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

		c := litecache.New[int](ctx)
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

		c := litecache.New[int](ctx)
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

		c := litecache.New[int](ctx)
		c.SetTtl("foo", 10, litecache.DefaultTtlCheckIntervals)
		assert.Equal(t, 1, c.Count())
		assert.Equal(t, 1, c.CountPrecise())

		v, found := c.Get("foo")
		assert.True(t, found)
		assert.Equal(t, 10, v)

		time.Sleep(litecache.DefaultTtlCheckIntervals - 100*time.Millisecond)

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 10, v)
		}

		time.Sleep(101 * time.Millisecond)

		{
			v, found := c.Get("foo")
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}

		assert.Equal(t, 0, c.Count())
		assert.Equal(t, 0, c.CountPrecise())

		// reset the key foo
		assert.True(t, c.SetNxTtl("foo", 25, 2*time.Second))

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 25, v)
			assert.Equal(t, 1, c.Count())
		}
	})
}

func TestNewWithConfig(t *testing.T) {
	t.Parallel()

	t.Run("0 shard number", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := litecache.NewDefaultConfig[float32]().WithShards(0)

		c, err := litecache.NewWithConfig[float32](ctx, cfg)
		require.Error(t, err)
		require.True(t, errors.Is(err, litecache.ErrInvalidConfig))
		require.Nil(t, c)
	})

	t.Run("negative shard number", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := litecache.NewDefaultConfig[float32]().WithShards(-1)

		c, err := litecache.NewWithConfig[float32](ctx, cfg)
		require.Error(t, err)
		require.True(t, errors.Is(err, litecache.ErrInvalidConfig))
		require.Nil(t, c)
	})

	t.Run("custom shard number", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := litecache.NewDefaultConfig[float32]().WithShards(10)

		c, err := litecache.NewWithConfig[float32](ctx, cfg)
		require.NoError(t, err)

		var wg sync.WaitGroup
		const iterations = 100_000

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), float32(i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Count())

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				c.Set(fmt.Sprintf("key:%d", i), float32(i))
			}(i)
		}

		wg.Wait()
		assert.Equal(t, iterations, c.Count())

		for i := 0; i < iterations; i++ {
			v, found := c.Get(fmt.Sprintf("key:%d", i))
			assert.True(t, found)
			assert.Equal(t, float32(i), v)
		}
	})

	t.Run("with custom on evict func", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		copyToCache := litecache.New[float32](ctx)

		cfg := litecache.NewDefaultConfig[float32]().
			WithShards(10).
			WithTtlChecksInterval(50 * time.Millisecond).
			WithOnEvict(func(key string, value float32) {
				copyToCache.Set(key, value)
			})

		originalCache, err := litecache.NewWithConfig[float32](ctx, cfg)
		require.NoError(t, err)

		var wg sync.WaitGroup
		const iterations = 100_000

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				originalCache.SetTtl(fmt.Sprintf("key:%d", i), float32(i), 75*time.Millisecond)
			}(i)
		}

		time.Sleep(200 * time.Millisecond)

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				key := fmt.Sprintf("key:%d", i)
				expectedValue := float32(i)
				v, found := copyToCache.Get(key)
				assert.True(t, found)
				assert.Equal(t, expectedValue, v)
			}(i)
		}

		wg.Wait()
	})
}

func TestCache_SetExTtl(t *testing.T) {
	t.Parallel()

	t.Run("try set non existent key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[float32](ctx)
		assert.False(t, c.SetEx("non:existent:key", 34))

		v, found := c.Get("non:existent:key")
		assert.False(t, found)
		assert.Equal(t, float32(0), v)
	})

	t.Run("try set non existent key with ttl", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[float32](ctx)
		assert.False(t, c.SetExTtl("non:existent", 34, 1*time.Second))

		v, found := c.Get("non:existent:key")
		assert.False(t, found)
		assert.Equal(t, float32(0), v)
	})

	t.Run("try set existing value with ttl", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[float32](ctx)
		c.Set("existent:key", 34)

		assert.True(t, c.SetExTtl("existent:key", 35, 1*time.Second))

		{
			v, found := c.Get("existent:key")
			assert.True(t, found)
			assert.Equal(t, float32(35), v)
		}

		time.Sleep(1 * time.Second)

		{
			v, found := c.Get("existent:key")
			assert.False(t, found)
			assert.Equal(t, float32(0), v)
		}
	})
}

func TestCache_Remove(t *testing.T) {
	t.Parallel()

	t.Run("remove existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const N = 1_000_000

		c := litecache.New[int](ctx)
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

		{
			v, found := c.Get("key:10")
			assert.True(t, found)
			assert.Equal(t, 10, v)
		}

		assert.True(t, c.Remove("key:10"))

		{
			v, found := c.Get("key:10")
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}
	})

	t.Run("remove and get existing key", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const N = 1_000_000

		c := litecache.New[int](ctx)
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

		{
			v, found := c.Get("key:10")
			assert.True(t, found)
			assert.Equal(t, 10, v)
		}

		{
			v, found := c.GetAndRemove("key:10")
			assert.True(t, found)
			assert.Equal(t, 10, v)
		}

		{
			v, found := c.Get("key:10")
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}
	})
}

func TestCache_GetAndSetEx(t *testing.T) {
	t.Parallel()

	t.Run("get and replace", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx)
		c.Set("foo", 3)
		assert.Equal(t, 1, c.Count())

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 3, v)
		}

		{
			v, found := c.GetAndSetEx("foo", 5)
			assert.True(t, found)
			assert.Equal(t, 3, v)
		}

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 5, v)
		}

		{
			v, found := c.GetAndSetEx("bar", 5)
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}
	})

	t.Run("get and replace with ttl", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx)
		c.Set("foo", 3)
		assert.Equal(t, 1, c.Count())

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 3, v)
		}

		{
			v, found := c.GetAndSetExTtl("foo", 5, 100*time.Millisecond)
			assert.True(t, found)
			assert.Equal(t, 3, v)
		}

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 5, v)
		}

		{
			v, found := c.GetAndSetEx("bar", 5)
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}

		time.Sleep(101 * time.Millisecond)

		{
			v, found := c.Get("foo")
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}
	})

	t.Run("get and replace expired value", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx)
		c.SetTtl("foo", 3, 200*time.Millisecond)
		assert.Equal(t, 1, c.Count())

		{
			v, found := c.Get("foo")
			assert.True(t, found)
			assert.Equal(t, 3, v)
		}

		time.Sleep(201 * time.Millisecond)

		{
			v, found := c.GetAndSetEx("foo", 5)
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}

		{
			v, found := c.Get("foo")
			assert.False(t, found)
			assert.Equal(t, 0, v)
		}
	})
}

func TestCache_ForEach(t *testing.T) {
	t.Parallel()

	t.Run("many keys sync", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		const N = 1_000_000

		c := litecache.New[int](ctx)
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

		c := litecache.New[int](ctx)
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

	t.Run("race", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		var wg sync.WaitGroup

		const N = 1_000_000
		c := litecache.New[int](ctx)

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := 1; i <= N; i++ {
				c.SetTtl(fmt.Sprintf("key:%d", i), i, 20*time.Millisecond)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := N; i >= 0; i-- {
				c.SetTtl(fmt.Sprintf("key:%d", i), i, 30*time.Millisecond)
			}
		}()

		time.Sleep(100 * time.Millisecond)
		c.SetNx("key:foo", 100)
		c.SetNx("key:bar", 200)

		wg.Add(1)
		go func() {
			defer wg.Done()

			totalFound := 0
			res := make(map[string]int)
			c.ForEach(func(k string, v int) {
				totalFound++
				res[k] = v
			})
			assert.GreaterOrEqual(t, totalFound, 2)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			for i := N; i <= N+N; i++ {
				c.SetNx(fmt.Sprintf("key:%d", i), i)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()

			totalFound := 0
			c.ForEach(func(k string, v int) {
				totalFound++
			})
			assert.GreaterOrEqual(t, totalFound, totalFound)
		}()

		wg.Wait()
	})
}

func TestCache_Transform(t *testing.T) {
	t.Parallel()

	t.Run("transform single value", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		c := litecache.New[int](ctx)
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

		c := litecache.New[int](ctx)
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
