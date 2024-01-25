# LiteCache
## Lite embedded in-memory cache library to use in GO applications

## API
```go
// sets key value pair.
// it will update the value if key already exists in the cache and has not expired.
Set(key string, value T)
```