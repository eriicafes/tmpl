package tmpl

import (
	"context"
	"sync"
)

// Async represents data which will be available in the future or an error.
type Async[T, E any] interface {
	// Ok resolves an Async with the data.
	// Ok or Err should be called exactly once.
	Ok(data T)

	// Err resolves an Async with the error.
	// Ok or Err should be called exactly once.
	Err(err E)

	asyncValue
}

type asyncValue interface {
	get(context.Context) (streamData, bool)
	getStored() (streamData, bool)
}

// NewAsync initializes a new Async in its pending state.
func NewAsync[T, E any]() Async[T, E] {
	return &async[T, E]{done: make(chan struct{})}
}

// Go initializes an Async and runs start in a new goroutine.
// start must call exactly one of Ok or Err.
func Go[T, E any](start func(Async[T, E])) Async[T, E] {
	value := NewAsync[T, E]()
	go start(value)
	return value
}

type async[T, E any] struct {
	mu    sync.RWMutex
	done  chan struct{}
	data  streamData
	isset bool
}

// Ok sets the stream data to a success value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *async[T, E]) Ok(data T) {
	a.resolve(streamData{ok: true, data: data})
}

// Err sets the stream data to an error value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *async[T, E]) Err(err E) {
	a.resolve(streamData{ok: false, data: err})
}

func (a *async[T, E]) resolve(data streamData) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.isset {
		panic("tmpl: Async resolved more than once")
	}
	a.data, a.isset = data, true
	close(a.done)
}

// get returns the stream data, blocking until the value resolves or ctx is cancelled.
func (a *async[T, E]) get(ctx context.Context) (streamData, bool) {
	select {
	case <-a.done:
		return a.getStored()
	case <-ctx.Done():
		return streamData{}, false
	}
}

// getStored returns the stream data and a boolean indicating the stream data has been set.
func (a *async[T, E]) getStored() (streamData, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data, a.isset
}
