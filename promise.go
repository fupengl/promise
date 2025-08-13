package promise

import (
	"context"
	"errors"
	"sync"
	"time"
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
	state    State
	value    T
	err      error
	handlers []*handler[T]
	mu       sync.RWMutex
}

// handler structure for processing
type handler[T any] struct {
	onFulfilled func(T) interface{}
	onRejected  func(error) interface{}
	next        *Promise[interface{}]
}

// New creates a new Promise
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	p := &Promise[T]{
		state:    Pending,
		handlers: make([]*handler[T], 0),
	}

	// Execute executor asynchronously
	go func() {
		defer func() {
			if r := recover(); r != nil {
				if err, ok := r.(error); ok {
					p.reject(err)
				} else {
					p.reject(errors.New("panic occurred"))
				}
			}
		}()

		executor(p.resolve, p.reject)
	}()

	return p
}

// resolve fulfills the Promise
func (p *Promise[T]) resolve(value T) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != Pending {
		return
	}

	p.state = Fulfilled
	p.value = value

	// Execute all fulfilled handlers
	for _, h := range p.handlers {
		if h.onFulfilled != nil {
			go func(h *handler[T]) {
				result := h.onFulfilled(value)
				if h.next != nil {
					h.next.resolve(result)
				}
			}(h)
		}
	}
}

// reject rejects the Promise
func (p *Promise[T]) reject(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.state != Pending {
		return
	}

	p.state = Rejected
	p.err = err

	// Execute all rejected handlers
	for _, h := range p.handlers {
		if h.onRejected != nil {
			go func(h *handler[T]) {
				result := h.onRejected(err)
				if h.next != nil {
					h.next.resolve(result)
				}
			}(h)
		} else if h.next != nil {
			// If no rejected handler, pass error to next Promise
			h.next.reject(err)
		}
	}
}

// Then adds fulfilled and rejected handlers
func (p *Promise[T]) Then(onFulfilled func(T) interface{}, onRejected func(error) interface{}) *Promise[interface{}] {
	next := &Promise[interface{}]{
		state:    Pending,
		handlers: make([]*handler[interface{}], 0),
	}

	h := &handler[T]{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		next:        next,
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If Promise is already completed, execute handlers immediately
	if p.state == Fulfilled {
		if onFulfilled != nil {
			go func() {
				result := onFulfilled(p.value)
				next.resolve(result)
			}()
		} else {
			next.resolve(p.value)
		}
	} else if p.state == Rejected {
		if onRejected != nil {
			go func() {
				result := onRejected(p.err)
				next.resolve(result)
			}()
		} else {
			next.reject(p.err)
		}
	} else {
		// Promise is still pending, add handler
		p.handlers = append(p.handlers, h)
	}

	return next
}

// Catch adds a rejected handler
func (p *Promise[T]) Catch(onRejected func(error) interface{}) *Promise[interface{}] {
	return p.Then(nil, onRejected)
}

// Finally adds a finally handler
func (p *Promise[T]) Finally(onFinally func()) *Promise[T] {
	next := &Promise[T]{
		state:    Pending,
		handlers: make([]*handler[T], 0),
	}

	p.Then(
		func(value T) interface{} {
			onFinally()
			next.resolve(value)
			return nil
		},
		func(err error) interface{} {
			onFinally()
			next.reject(err)
			return nil
		},
	)

	return next
}

// Await waits for Promise completion and returns result
func (p *Promise[T]) Await() (T, error) {
	for {
		p.mu.RLock()
		if p.state != Pending {
			p.mu.RUnlock()
			break
		}
		p.mu.RUnlock()
		time.Sleep(time.Millisecond)
	}

	if p.state == Fulfilled {
		return p.value, nil
	}
	var zero T
	return zero, p.err
}

// AwaitWithContext waits for Promise completion using context
func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	for {
		select {
		case <-ctx.Done():
			var zero T
			return zero, ctx.Err()
		default:
			p.mu.RLock()
			if p.state != Pending {
				p.mu.RUnlock()
				goto done
			}
			p.mu.RUnlock()
			time.Sleep(time.Millisecond)
		}
	}

done:
	if p.state == Fulfilled {
		return p.value, nil
	}
	var zero T
	return zero, p.err
}

// State gets the Promise state
func (p *Promise[T]) State() State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// IsPending checks if Promise is in pending state
func (p *Promise[T]) IsPending() bool {
	return p.State() == Pending
}

// IsFulfilled checks if Promise is fulfilled
func (p *Promise[T]) IsFulfilled() bool {
	return p.State() == Fulfilled
}

// IsRejected checks if Promise is rejected
func (p *Promise[T]) IsRejected() bool {
	return p.State() == Rejected
}
