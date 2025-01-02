package tmpl

// AsyncValue represents data which will be available in the future or an error.
type AsyncValue[T, E any] interface {
	// Ok resolves an AsyncValue with the data.
	// Ok or Err should be called exactly once.
	Ok(data T)

	// Err resolves an AsyncValue with the error.
	// Ok or Err should be called exactly once.
	Err(err E)

	asyncValueRenderer
}

type asyncValueRenderer interface {
	renderer() *renderer
	readyChan() chan struct{}
	get() streamData
	getCached() (streamData, bool)
}

// AsyncValue initializes a new AsyncValue in it's pending state.
func NewAsyncValue[T, E any](r Renderer) AsyncValue[T, E] {
	return &asyncValue[T, E]{
		r:  r,
		ch: make(chan struct{}),
	}
}

type asyncValue[T, E any] struct {
	r     Renderer
	ch    chan struct{}
	data  streamData
	ready bool
}

func (a *asyncValue[T, E]) Ok(data T) {
	a.data, a.ready = streamData{true, data}, true
	close(a.ch)
}

func (a *asyncValue[T, E]) Err(err E) {
	a.data, a.ready = streamData{false, err}, true
	close(a.ch)
}

func (a *asyncValue[T, E]) renderer() *renderer {
	return a.r.Unwrap()
}

func (a *asyncValue[T, E]) readyChan() chan struct{} {
	return a.ch
}

func (a *asyncValue[T, E]) get() streamData {
	<-a.ch
	return a.data
}

func (a *asyncValue[T, E]) getCached() (streamData, bool) {
	return a.data, a.ready
}
