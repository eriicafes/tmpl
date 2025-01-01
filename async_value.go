package tmpl

// AsyncValue represents data which will be available in the future or an error.
// It is implemented using a non-buffered channel.
//
// When using sync rendering, calls to Ok or Err will block until the template is rendered
// which will wait until all earlier rendered templates in the markup are resolved and rendered.
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
	renderer() (Renderer, chan streamData)
}

// AsyncValue initializes a new AsyncValue in it's pending state.
func NewAsyncValue[T, E any](r Renderer) AsyncValue[T, E] {
	return &asyncValue[T, E]{r, make(chan streamData)}
}

type asyncValue[T, E any] struct {
	r  Renderer
	ch chan streamData
}

func (a *asyncValue[T, E]) Ok(data T) {
	a.ch <- streamData{true, data}
}

func (a *asyncValue[T, E]) Err(err E) {
	a.ch <- streamData{false, err}
}

func (a *asyncValue[T, E]) renderer() (Renderer, chan streamData) {
	return a.r, a.ch
}
