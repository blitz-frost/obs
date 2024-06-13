// Package obs provides code instrumentation.
package obs

import "sync"

// A Loader can safely obtain values for inspection.
type Loader interface {
	Load() any
}

// A Map groups and provides access to a set of Values.
//
// Its methods are concurrent safe.
type Map struct {
	values map[any]Value
	mux    sync.Mutex
}

func MapMake() *Map {
	return &Map{
		values: make(map[any]Value),
	}
}

func (x *Map) Delete(key any) {
	x.mux.Lock()
	delete(x.values, key)
	x.mux.Unlock()
}

func (x *Map) Get(key any) (Value, bool) {
	x.mux.Lock()
	o, ok := x.values[key]
	x.mux.Unlock()
	return o, ok
}

// Range calls the given function with the labels and loaded values of all members.
func (x *Map) Range(fn func(string, any)) {
	x.mux.Lock()
	for _, v := range x.values {
		fn(v.Label, v.Load())
	}
	x.mux.Unlock()
}

func (x *Map) Set(key any, val Value) {
	x.mux.Lock()
	x.values[key] = val
	x.mux.Unlock()
}

// A Sampler accepts samples in a finite queue, and processes them in a dedicated goroutine.
// If the sample queue would overflow, emits a warning and discards all subsequent samples.
type Sampler[S any, T any] struct {
	Final    func(*S)    // called when the last sample has been processed, if non-nil
	First    func(*S, T) // called on the first sample, before the normal sampling function, if non-nil
	Overflow func()      // called when a queue overflow occurs, if non-nil

	state S

	sampleChan chan T
	sampleFunc func(*S, T)

	inactive bool
	stop     bool
}

func SamplerMake[S any, T any](queueSize int, sampleFunc func(*S, T)) *Sampler[S, T] {
	return &Sampler[S, T]{
		sampleChan: make(chan T, queueSize),
		sampleFunc: sampleFunc,
		inactive:   true,
	}
}

// Sample pushes a new sample for the Sampler to process.
// NoOp if the Sampler is inactive (closed or has overflowed).
func Sample[S any, T any](x *Sampler[S, T], v T) {
	if x.inactive {
		return
	}

	x.sampleChan <- v
	if x.stop {
		x.inactive = true
		close(x.sampleChan)

		if x.Overflow != nil {
			x.Overflow()
		}
	}
}

func Start[S any, T any](x *Sampler[S, T]) {
	x.inactive = false
	go loop(x)
}

// Stop terminates the active processing loop, if it exists.
// Must be called when the Sampler is no longer needed.
func Stop[S any, T any](x *Sampler[S, T]) {
	x.inactive = true
	close(x.sampleChan)
}

func loop[S any, T any](x *Sampler[S, T]) {
	if x.Final != nil {
		defer func() {
			x.Final(&x.state)
		}()
	}

	if x.First != nil {
		sample, ok := <-x.sampleChan
		if !ok {
			return
		}

		x.First(&x.state, sample)
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

type Value struct {
	Label string
	Loader
}
