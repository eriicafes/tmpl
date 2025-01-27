package tmpl

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
	get() streamData
	getCached() (streamData, bool)
	doneChan() chan struct{}
	renderer() Renderer
}

// AsyncValue initializes a new AsyncValue in it's pending state.
func NewAsyncValue[T, E any](r Renderer) AsyncValue[T, E] {
	return &asyncValue[T, E]{r: r, done: make(chan struct{})}
}

type asyncValue[T, E any] struct {
	r     Renderer
	done  chan struct{}
	data  streamData
	isset bool
}

// Ok sets the stream data to a success value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *asyncValue[T, E]) Ok(data T) {
	a.data, a.isset = streamData{true, data}, true
	close(a.done)
}

// Err sets the stream data to an error value and closes the channel.
// isset indicates the stream data has been set and is not the default value.
func (a *asyncValue[T, E]) Err(err E) {
	a.data, a.isset = streamData{false, err}, true
	close(a.done)
}

// get returns the stream data, blocking until the done channel is closed.
func (a *asyncValue[T, E]) get() streamData {
	<-a.done
	return a.data
}

// getCached returns the stream data and a boolean indicating the stream data has been set.
func (a *asyncValue[T, E]) getCached() (streamData, bool) { return a.data, a.isset }

// doneChan returns a done channel that will be closed once the stream data is set.
func (a *asyncValue[T, E]) doneChan() chan struct{} { return a.done }

// renderer returns the underlying renderer.
func (a *asyncValue[T, E]) renderer() Renderer { return a.r }
