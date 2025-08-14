package promise

import (
	"sync"
	"sync/atomic"
)

// Promise state enumeration
type State int

const (
	Pending State = iota
	Fulfilled
	Rejected
)

// Promise structure
type Promise[T any] struct {
	// Atomic state and value management
	state atomic.Value // State
	value atomic.Value // T
	err   atomic.Value // error

	// Protected by mutex
	handlers []*handler[T]
	mu       sync.RWMutex
	done     chan struct{} // Signal channel for completion

	// Associated manager
	manager *PromiseMgr
}

// Helper function: safely get state
func (p *Promise[T]) getState() State {
	if state, ok := p.state.Load().(State); ok {
		return state
	}
	return Pending
}

// Helper function: safely set state
func (p *Promise[T]) setState(state State) bool {
	// Initialize if needed, then use CompareAndSwap
	if p.state.Load() == nil {
		p.state.Store(Pending)
	}

	// Try to transition from Pending to target state
	if p.state.CompareAndSwap(Pending, state) {
		return true
	}

	// If transition failed, check current state
	currentState := p.state.Load().(State)
	return currentState == state // Return true if already in target state
}

// Helper function: safely get value
func (p *Promise[T]) getValue() (T, bool) {
	if value, ok := p.value.Load().(T); ok {
		return value, true
	}
	var zero T
	return zero, false
}

// Helper function: safely set value
func (p *Promise[T]) setValue(value T) {
	p.value.Store(value)
}

// Helper function: safely get error
func (p *Promise[T]) getError() (error, bool) {
	if err, ok := p.err.Load().(error); ok {
		return err, true
	}
	return nil, false
}

// Helper function: safely set error
func (p *Promise[T]) setError(err error) {
	p.err.Store(err)
}

// Handler structure for processing
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
