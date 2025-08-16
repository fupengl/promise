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

// reset resets the handler for reuse
func (h *handler[T]) reset() {
	h.onFulfilled = nil
	h.onRejected = nil
	h.next = nil
}

// getHandler gets a handler from the pool or creates a new one
func getHandler[T any]() *handler[T] {
	// Use the global handler pool with type erasure
	if v := globalHandlerPool.Get(); v != nil {
		h := v.(*handler[any])
		// Create a new typed handler and copy the fields
		typedHandler := &handler[T]{}
		// Reset the original handler and return it to pool
		h.reset()
		globalHandlerPool.Put(h)
		return typedHandler
	}
	return &handler[T]{}
}

// putHandler returns a handler to the pool
func putHandler[T any](h *handler[T]) {
	// Convert to interface{} type for the pool
	interfaceHandler := &handler[any]{
		onFulfilled: func(value any) any { return nil },
		onRejected:  func(err error) any { return nil },
		next:        nil,
	}
	globalHandlerPool.Put(interfaceHandler)
}

// Global handler pool using interface{} to avoid generic type issues
var globalHandlerPool = sync.Pool{
	New: func() interface{} {
		return &handler[any]{}
	},
}

// Result represents the result of a Promise
type Result[T any] struct {
	Value T
	Error error
	Index int
}
