package tmpl

import (
	"context"
	"sync"
)

// AsyncValue represents data which will be available in the future or an error.
type AsyncValue[T, E any] interface {
	// Ok resolves an AsyncValue with the data.
	// Ok or Err should be called exactly once.
	Ok(data T)

	// Err resolves an AsyncValue with the error.
	// Ok or Err should be called exactly once.
	Err(err E)

	asyncValuer
}

type asyncValuer interface {
	get(context.Context) (streamData, bool)
	getCached() (streamData, bool)
}

// AsyncValue initializes a new AsyncValue in it's pending state.
func NewAsyncValue[T, E any]() AsyncValue[T, E] {
	return &asyncValue[T, E]{done: make(chan struct{})}
}

// Go initializes an AsyncValue and runs start in a new goroutine.
// start must call exactly one of Ok or Err.
func Go[T, E any](start func(AsyncValue[T, E])) AsyncValue[T, E] {
	value := NewAsyncValue[T, E]()
	go start(value)
	return value
}

type asyncValue[T, E any] struct {
	mu    sync.RWMutex
	done  chan struct{}
	data  streamData
	isset bool
}

// Ok sets the stream data to a success value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *asyncValue[T, E]) Ok(data T) {
	a.resolve(streamData{ok: true, data: data})
}

// Err sets the stream data to an error value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *asyncValue[T, E]) Err(err E) {
	a.resolve(streamData{ok: false, data: err})
}

func (a *asyncValue[T, E]) resolve(data streamData) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.isset {
		panic("tmpl: AsyncValue resolved more than once")
	}
	a.data, a.isset = data, true
	close(a.done)
}

// get returns the stream data, blocking until the value resolves or ctx is cancelled.
func (a *asyncValue[T, E]) get(ctx context.Context) (streamData, bool) {
	select {
	case <-a.done:
		return a.getCached()
	case <-ctx.Done():
		return streamData{}, false
	}
}

// getCached returns the stream data and a boolean indicating the stream data has been set.
func (a *asyncValue[T, E]) getCached() (streamData, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data, a.isset
}
