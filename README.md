# LiteCache
## Lite embedded in-memory cache library to use in GO applications

## Initialization
#### With default config and int as stored values type
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

c := litecache.New[int](ctx)
```

#### With custom config
```go
cfg := litecache.NewDefaultConfig[string]().
			WithShards(20).
			WithOnEvict(func(key string, value string) {
				log.Printf("evicted key %s value %s", key, value)
			})

c, err := litecache.NewWithConfig[string](ctx, cfg)
```
NewWithConfig may return litecache.ErrInvalidConfig when configuration is invalid

### Usage
```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

c := litecache.New[int](ctx)

// set key foo with value 10 for 2 seconds
{
    c.SetTtl("foo", 10, 2*time.Second)
    c.Count() // 1
    v, found := c.Get("foo") // 10, true
}

time.Sleep(3 * time.Second)

{
    v, found := c.Get("foo") // 0, false
}

c.SetEx("non:existent:key", 34) // false since that key does not exist
c.SetEx("foo", 34) // false since that key has expired
c.SetNx("foo", 34) // true since key has expired 
                   // and that is the same as if it has never existed

```

## API
Set - sets the key value pair. 
It will update the value if key already exists in the cache and has not expired.
```go
func (c *Cache[T]) Set(key string, value T)
```

Get - returns value for a key, if it exists and has not expired
zero value is returned if the key was not found or has expired
```go
func (c *Cache[T]) Get(key string) (T, bool) {
```

SetTtl - sets key value pair with ttl.
it will update the value if key already exists in the cache and has not expired.
```go
func (c *Cache[T]) SetTtl(key string, value T, ttl time.Duration)
```

SetNx - sets key value pair only if key does not exist in the cache or has expired.
if the key value pair was set successfully it returns true
```go
func (c *Cache[T]) SetNx(key string, value T) bool
```

SetNxTtl - sets key value only if key does not exist in the cache or has expired.
ttl expected to be given as a last parameter.
if the key value pair was set successfully it returns true
```go
func (c *Cache[T]) SetNxTtl(key string, value T, ttl time.Duration) bool
```

SetEx - updates key value pair if key already exists and not expired in the cache.
if value was updated, returns true
```go
func (c *Cache[T]) SetEx(key string, value T) bool
```

SetExTtl - updates key value pair if key already exists and not expired in the cache.
ttl expected to be given as a last parameter.
if value was updated, returns true
```go
func (c *Cache[T]) SetExTtl(key string, value T, ttl time.Duration) bool
```

Remove - removes the value from cache if present
returns true if key was found and false if it was not
```go
func (c *Cache[T]) Remove(key string) bool
```

GetAndRemove - removes the value from cache if present
returns value and boolean true if key was found and zero value and boolean false if it was not
```go
func (c *Cache[T]) GetAndRemove(key string) (T, bool)
```

Count returns the number of in the cache keys.
This method is eventually consistent, it might get delayed updates when keys expire.
```go
func (c *Cache[T]) Count() int
```

GetAndSetExTtl sets the value for existing key, only if it exists in the cache
it will return the old value and true if the key found in cache and zero value and false if not found or expired
ttl expiration is expected to be given as a last parameter.
```go
func (c *Cache[T]) GetAndSetExTtl(key string, value T, ttl time.Duration) (T, bool)
```

GetAndSetEx sets the value for existing key, only if it exists in the cache
it will return the old value and true if the key found in cache and zero value and false if not found or expired
```go
func (c *Cache[T]) GetAndSetEx(key string, value T) (T, bool)
```

Transform can change the value of the given key atomically
it does not modify the ttl of the key
```go
func (c *Cache[T]) Transform(key string, effector func(value T) T) bool
```

ForEach iterates over all the keys and values that are not expired in the cache
the method uses mutex to lock the content of the cache for reading
```go
func (c *Cache[T]) ForEach(fn func(k string, v T))
```






