// Package obs provides code instrumentation.
package obs

// Observer is the foundational type of this package.
// It gathers samples in a finite queue, and processes these samples in a dedicated goroutine.
// If the sample queue would overflow, emits a warning and discards all subsequent samples.
type Observer[S any, T any] struct {
	OnFinal    func()      // called when the last sample has been processed, if non-nil
	OnFirst    func(*S, T) // called on the first sample, before the normal sampling function, if non-nil
	OnOverflow func()      // called when a queue overflow occurs, if non-nil

	state S

	sampleChan chan T
	sampleFunc func(*S, T)

	inactive bool
	stop     bool
}

func NewObserver[S any, T any](queueSize int, sampleFunc func(*S, T)) *Observer[S, T] {
	return &Observer[S, T]{
		sampleChan: make(chan T, queueSize),
		sampleFunc: sampleFunc,
		inactive:   true,
	}
}

// Sample pushes a new sample for the Observer to process.
// NoOp if the Observer is inactive (closed or has overflowed).
func Sample[S any, T any](x *Observer[S, T], v T) {
	if x.inactive {
		return
	}

	x.sampleChan <- v
	if x.stop {
		x.inactive = true
		close(x.sampleChan)

		if x.OnOverflow != nil {
			x.OnOverflow()
		}
	}
}

func Start[S any, T any](x *Observer[S, T]) {
	x.inactive = false
	go loop(x)
}

// Stop terminates the active processing loop, if it exists.
// Must be called when the Observer is no longer needed.
func Stop[S any, T any](x *Observer[S, T]) {
	x.inactive = true
	close(x.sampleChan)
}

func loop[S any, T any](x *Observer[S, T]) {
	if x.OnFinal != nil {
		defer x.OnFinal()
	}

	if x.OnFirst != nil {
		sample, ok := <-x.sampleChan
		if !ok {
			return
		}

		x.OnFirst(&x.state, sample)
		x.sampleFunc(&x.state, sample)
	}

	for sample := range x.sampleChan {
		x.sampleFunc(&x.state, sample)

		if len(x.sampleChan) == cap(x.sampleChan) {
			// we have reached overflow
			x.stop = true
		}
	}
}
