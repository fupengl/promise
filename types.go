package promise

import "sync"

// Promise state enumeration
type State int

const (
	Pending State = iota
	Fulfilled
	Rejected
)

// Promise structure
type Promise[T any] struct {
	state    State
	value    T
	err      error
	handlers []*handler[T]
	mu       sync.RWMutex
	done     chan struct{} // Signal channel for completion
}

// handler structure for processing
type handler[T any] struct {
	onFulfilled func(T) any
	onRejected  func(error) any
	next        *Promise[any]
}

// Result represents the result of a Promise
type Result[T any] struct {
	Value T
	Error error
	Index int
}
