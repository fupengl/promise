package promise

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type State int

const (
	Pending State = iota
	Fulfilled
	Rejected
)

type ErrorType int

const (
	RejectionError ErrorType = iota // Promise rejected
	PanicError                      // Panic occurred in callback
	TimeoutError                    // Promise timeout
	ManagerError                    // Manager-related errors
)

type PromiseError struct {
	Message       string      // Human-readable error message
	Cause         error       // Original error that caused this
	Type          ErrorType   // Type of error
	Value         interface{} // Original panic value (for non-error panics)
	OriginalError error       // Original error being processed (for error callbacks)
}

// Error implements the error interface
func (e *PromiseError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	if e.Value != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Value)
	}
	return e.Message
}

// Unwrap returns the cause error for error wrapping
func (e *PromiseError) Unwrap() error {
	return e.Cause
}

// Is implements error comparison
func (e *PromiseError) Is(target error) bool {
	if target == nil {
		return e == nil
	}

	if promiseErr, ok := target.(*PromiseError); ok {
		return e.Type == promiseErr.Type
	}

	return false
}

// As implements error type assertion
func (e *PromiseError) As(target interface{}) bool {
	if target == nil {
		return false
	}

	switch t := target.(type) {
	case **PromiseError:
		*t = e
		return true
	case *ErrorType:
		*t = e.Type
		return true
	}

	return false
}

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
	// Use atomic operation to ensure only one goroutine can set the state
	// First, try to initialize to Pending if not already set
	for {
		currentState := p.state.Load()
		if currentState == nil {
			// Try to initialize to Pending
			if p.state.CompareAndSwap(nil, Pending) {
				break // Successfully initialized
			}
			continue // Another goroutine initialized, retry
		}
		break // Already initialized
	}

	// Now try to transition from Pending to target state
	return p.state.CompareAndSwap(Pending, state)
}

// Helper function: safely get value
func (p *Promise[T]) getValue() (T, bool) {
	if ptr, ok := p.value.Load().(*T); ok && ptr != nil {
		return *ptr, true
	}
	var zero T
	return zero, false
}

// Helper function: safely set value
func (p *Promise[T]) setValue(value T) {
	// Use pointer wrapper to safely store nil values in atomic.Value
	// Allocate a new value on heap to avoid issues with local variable addresses
	ptr := new(T)
	*ptr = value
	p.value.Store(ptr)
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

type handler[T any] struct {
	onFulfilled func(T) any
	onRejected  func(error) any
	next        *Promise[any]
}

type Result[T any] struct {
	Value T
	Error error
	Index int
}

// Predefined errors for common scenarios
var (
	// ErrManagerStopped is returned when trying to use a stopped manager
	ErrManagerStopped = &PromiseError{
		Message: "manager is stopped",
		Type:    ManagerError,
	}
)
