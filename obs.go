// Package obs provides code instrumentation.
package obs

import (
	"sync"
	"sync/atomic"
)

// A Loader can safely obtain values for inspection.
type Loader func() any

// A Map groups and provides access to a set of Values.
//
// Its methods are concurrent safe.
type Map struct {
	_values map[any]Value
	_mux    sync.Mutex
}

func MapMake() *Map {
	return &Map{
		_values: make(map[any]Value),
	}
}

func (x *Map) Delete(key any) {
	x._mux.Lock()
	delete(x._values, key)
	x._mux.Unlock()
}

func (x *Map) Get(key any) (Value, bool) {
	x._mux.Lock()
	o, ok := x._values[key]
	x._mux.Unlock()
	return o, ok
}

// Range calls the given function with the labels and loaded values of all members.
func (x *Map) Range(fn func(Value)) {
	x._mux.Lock()
	for _, v := range x._values {
		fn(v)
	}
	x._mux.Unlock()
}

// The key must be a comparable type.
//
// The Value label isn't used as a key, because that would force the user to store it, if they want to delete it later.
// This way, a simple pointer can be used instead, for example.
func (x *Map) Set(key any, val Value) {
	x._mux.Lock()
	x._values[key] = val
	x._mux.Unlock()
}

// A Series captures a target number of samples, once the Load() method is called.
type Series[T any] struct {
	Samples int // how many samples will be gathered per Load()

	_values []T
	_active atomic.Bool
	_wait   chan struct{}
}

// Load enables sample storing and blocks until the target number is obtained.
// The returned value is a []T.
//
// Not concurrent safe with itself.
func (x *Series[T]) Load() any {
	x._values = make([]T, 0, x.Samples)
	x._wait = make(chan struct{})
	x._active.Store(true)

	<-x._wait

	return x._values
}

// Not concurrent safe with itself.
func (x *Series[T]) Store(v T) {
	if !x._active.Load() {
		return
	}

	x._values = append(x._values, v)

	if len(x._values) == cap(x._values) {
		x._active.Store(false)
		close(x._wait)
	}
}

type Value struct {
	Label string
	Loader
}
