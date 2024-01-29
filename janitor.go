package litecache

import (
	"context"
	"time"
)

type janitor[T any] struct {
	ctx      context.Context
	interval time.Duration
}

func newJanitor[T any](ctx context.Context, runEvery time.Duration) *janitor[T] {
	return &janitor[T]{
		ctx:      ctx,
		interval: runEvery,
	}
}

func (j *janitor[T]) runOn(s *shard[T], onAfterIteration func(key string, value T)) {
	go func() {
		tick := time.NewTicker(j.interval)

		for {
			select {
			case <-j.ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				s.cleanExpired(onAfterIteration)
			}
		}
	}()
}
