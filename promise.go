package promise

import (
	"context"
	"errors"
)

// New creates a new Promise using the default manager
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	return NewWithMgr(GetDefaultMgr(), executor)
}

// NewWithManager creates a new Promise using the specified manager
func NewWithMgr[T any](manager *PromiseMgr, executor func(resolve func(T), reject func(error))) *Promise[T] {
	p := &Promise[T]{
		manager: manager,
	}

	// Initialize atomic values
	p.state.Store(Pending)
	// handlers remain lazy initialized, only created when Then is called

	// Execute executor asynchronously using the manager
	manager.scheduleExecutor(func() {
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
	})

	return p
}

// resolve fulfills the Promise
func (p *Promise[T]) resolve(value T) {
	if !p.setState(Fulfilled) {
		return
	}

	p.setValue(value)

	p.mu.Lock()
	if p.done == nil {
		p.done = make(chan struct{})
	}
	close(p.done)

	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

	if handlers != nil {
		for _, h := range handlers {
			if h.onFulfilled != nil {
				handler := h.onFulfilled
				next := h.next
				val := value

				p.manager.scheduleMicrotask(func() {
					safeCallback(handler, val, next)
				})
			}
		}
	}
}

// reject rejects the Promise
func (p *Promise[T]) reject(err error) {
	if !p.setState(Rejected) {
		return
	}

	p.setError(err)

	p.mu.Lock()
	if p.done == nil {
		p.done = make(chan struct{})
	}
	close(p.done)

	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

	if handlers != nil {
		for _, h := range handlers {
			if h.onRejected != nil {
				handler := h.onRejected
				next := h.next
				errVal := err

				p.manager.scheduleMicrotask(func() {
					safeErrorCallback(handler, errVal, next)
				})
			} else if h.next != nil {
				next := h.next
				errVal := err

				p.manager.scheduleMicrotask(func() {
					next.reject(errVal)
				})
			}
		}
	}
}

// Then adds fulfilled and rejected handlers
func (p *Promise[T]) Then(onFulfilled func(T) any, onRejected func(error) any) *Promise[any] {
	next := &Promise[any]{
		done:    make(chan struct{}),
		manager: p.manager,
	}

	next.state.Store(Pending)

	h := &handler[T]{
		onFulfilled: onFulfilled,
		onRejected:  onRejected,
		next:        next,
	}

	state := p.getState()

	if state == Fulfilled {
		if onFulfilled != nil {
			value, _ := p.getValue()
			p.manager.scheduleMicrotask(func() {
				safeCallback(onFulfilled, value, next)
			})
		} else {
			value, _ := p.getValue()
			next.resolve(value)
		}
		return next
	}

	if state == Rejected {
		if onRejected != nil {
			err, _ := p.getError()
			p.manager.scheduleMicrotask(func() {
				safeErrorCallback(onRejected, err, next)
			})
		} else {
			err, _ := p.getError()
			next.reject(err)
		}
		return next
	}

	p.mu.Lock()
	if p.getState() == Pending {
		if p.handlers == nil {
			p.handlers = make([]*handler[T], 0, 4)
		}
		p.handlers = append(p.handlers, h)
	}
	p.mu.Unlock()

	return next
}

// Catch adds a rejected handler
func (p *Promise[T]) Catch(onRejected func(error) any) *Promise[any] {
	return p.Then(nil, onRejected)
}

// Finally adds a finally handler
func (p *Promise[T]) Finally(onFinally func()) *Promise[T] {
	next := &Promise[T]{
		done:    make(chan struct{}),
		manager: p.manager,
	}

	next.state.Store(Pending)

	// Create a handler that will execute the finally callback
	h := &handler[T]{
		onFulfilled: func(value T) any {
			safeFinallyCallback(onFinally, next, value, nil)
			return nil
		},
		onRejected: func(err error) any {
			value, _ := p.getValue()
			safeFinallyCallback(onFinally, next, value, err)
			return nil
		},
		next: nil, // We don't need to chain to another promise here
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// If Promise is already completed, execute handlers using microtask queue
	if p.getState() == Fulfilled {
		// Capture values to avoid race conditions
		handler := h.onFulfilled
		val, _ := p.getValue()

		p.manager.scheduleMicrotask(func() {
			handler(val)
		})
	} else if p.getState() == Rejected {
		// Capture values to avoid race conditions
		handler := h.onRejected
		errVal, _ := p.getError()

		p.manager.scheduleMicrotask(func() {
			handler(errVal)
		})
	} else {
		// Promise is still pending, add handler
		// Lazy initialize handlers slice
		if p.handlers == nil {
			p.handlers = make([]*handler[T], 0, 4)
		}
		p.handlers = append(p.handlers, h)
	}

	return next
}

// Await waits for Promise completion and returns result
func (p *Promise[T]) Await() (T, error) {
	if p.done == nil {
		p.done = make(chan struct{})
	}

	// Wait for completion signal instead of polling
	<-p.done

	if p.getState() == Fulfilled {
		value, _ := p.getValue()
		return value, nil
	}
	var zero T
	err, _ := p.getError()
	return zero, err
}

// AwaitWithContext waits for Promise completion using context
func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	if p.done == nil {
		p.done = make(chan struct{})
	}

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-p.done:
		// Promise completed
	}

	if p.getState() == Fulfilled {
		value, _ := p.getValue()
		return value, nil
	}
	var zero T
	err, _ := p.getError()
	return zero, err
}

// State gets the Promise state
func (p *Promise[T]) State() State {
	return p.getState()
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

// safeCallback wraps a callback function with panic recovery
func safeCallback[T any](callback func(T) any, value T, next *Promise[any]) {
	if next == nil {
		callback(value)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			var err error
			if e, ok := r.(error); ok {
				err = e
			} else {
				err = errors.New("panic occurred in callback")
			}
			next.reject(err)
		}
	}()

	result := callback(value)
	next.resolve(result)
}

// safeErrorCallback wraps an error callback function with panic recovery
func safeErrorCallback(callback func(error) any, err error, next *Promise[any]) {
	if next == nil {
		callback(err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = errors.New("panic occurred in error callback")
			}
			next.reject(panicErr)
		}
	}()

	result := callback(err)
	next.resolve(result)
}

// safeFinallyCallback wraps a finally callback function with panic recovery
func safeFinallyCallback[T any](callback func(), next *Promise[T], value T, err error) {
	if next == nil {
		callback()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = e
			} else {
				panicErr = errors.New("panic occurred in finally callback")
			}
			next.reject(panicErr)
			return
		}

		// Finally callback completed successfully, resolve with original value
		if err == nil {
			next.resolve(value)
		} else {
			next.reject(err)
		}
	}()

	callback()
}
