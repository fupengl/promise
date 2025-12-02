package promise

import (
	"context"
	"fmt"
)

// New creates a Promise using the default manager
func New[T any](executor func(resolve func(T), reject func(error))) *Promise[T] {
	return NewWithMgr(GetDefaultMgr(), executor)
}

// WithResolvers creates a Promise and returns resolve/reject functions
func WithResolvers[T any]() (*Promise[T], func(T), func(error)) {
	return WithResolversWithMgr[T](GetDefaultMgr())
}

// WithResolversWithMgr creates a Promise with specified manager
func WithResolversWithMgr[T any](manager *PromiseMgr) (*Promise[T], func(T), func(error)) {
	p := &Promise[T]{
		manager: manager,
		done:    make(chan struct{}),
	}

	p.state.Store(Pending)

	if manager.IsShutdown() {
		p.reject(&PromiseError{
			Message: "Promise creation failed: manager is shutdown",
			Cause:   ErrManagerStopped,
			Type:    RejectionError,
		})
		return p, p.resolve, p.reject
	}

	return p, p.resolve, p.reject
}

// NewWithMgr creates a Promise using the specified manager
func NewWithMgr[T any](manager *PromiseMgr, executor func(resolve func(T), reject func(error))) *Promise[T] {
	p := &Promise[T]{
		manager: manager,
		done:    make(chan struct{}),
	}

	p.state.Store(Pending)

	if manager.IsShutdown() {
		p.reject(&PromiseError{
			Message: "Promise creation failed: manager is shutdown",
			Cause:   ErrManagerStopped,
			Type:    RejectionError,
		})
		return p
	}

	if err := manager.scheduleExecutor(func() {
		defer func() {
			if r := recover(); r != nil {
				p.reject(createPanicError(r, "panic in executor"))
			}
		}()

		executor(p.resolve, p.reject)
	}); err != nil {
		p.reject(&PromiseError{
			Message: "failed to schedule executor",
			Cause:   err,
			Type:    RejectionError,
		})
	}

	return p
}

func (p *Promise[T]) resolve(value T) {
	if !p.setState(Fulfilled) {
		return
	}

	p.setValue(value)

	p.mu.Lock()
	close(p.done)
	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

	for _, h := range handlers {
		if h.onFulfilled != nil {
			p.manager.scheduleMicrotask(func() {
				safeCallback(h.onFulfilled, value, h.next)
			})
		}
	}
}

func (p *Promise[T]) reject(err error) {
	if !p.setState(Rejected) {
		return
	}

	finalErr := wrapErrorIfNeeded(err, "Promise rejected", RejectionError)
	p.setError(finalErr)

	p.mu.Lock()
	close(p.done)
	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

	for _, h := range handlers {
		if h.onRejected != nil {
			p.manager.scheduleMicrotask(func() {
				safeErrorCallback(h.onRejected, finalErr, h.next)
			})
		} else if h.next != nil {
			p.manager.scheduleMicrotask(func() {
				h.next.reject(finalErr)
			})
		}
	}
}

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
		value, _ := p.getValue()
		if onFulfilled != nil {
			p.manager.scheduleMicrotask(func() {
				safeCallback(onFulfilled, value, next)
			})
		} else {
			next.resolve(value)
		}
		return next
	}

	if state == Rejected {
		err, _ := p.getError()
		if onRejected != nil {
			p.manager.scheduleMicrotask(func() {
				safeErrorCallback(onRejected, err, next)
			})
		} else {
			next.reject(err)
		}
		return next
	}

	p.mu.Lock()
	if p.getState() == Pending {
		if p.handlers == nil {
			p.handlers = make([]*handler[T], 0, 2)
		}
		p.handlers = append(p.handlers, h)
	}
	p.mu.Unlock()

	return next
}

func (p *Promise[T]) Catch(onRejected func(error) any) *Promise[any] {
	return p.Then(nil, onRejected)
}

func (p *Promise[T]) Finally(onFinally func()) *Promise[T] {
	next := &Promise[T]{
		done:    make(chan struct{}),
		manager: p.manager,
	}

	next.state.Store(Pending)

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
		next: nil,
	}

	state := p.getState()

	if state == Fulfilled {
		val, _ := p.getValue()
		p.manager.scheduleMicrotask(func() {
			h.onFulfilled(val)
		})
	} else if state == Rejected {
		errVal, _ := p.getError()
		p.manager.scheduleMicrotask(func() {
			h.onRejected(errVal)
		})
	} else {
		p.mu.Lock()
		if p.handlers == nil {
			p.handlers = make([]*handler[T], 0, 2)
		}
		p.handlers = append(p.handlers, h)
		p.mu.Unlock()
	}

	return next
}

func (p *Promise[T]) Await() (T, error) {
	return p.AwaitWithContext(context.Background())
}

func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	// Fast path: check if already resolved
	if state := p.getState(); state != Pending {
		if state == Fulfilled {
			if value, ok := p.getValue(); ok {
				return value, nil
			}
		} else {
			if err, ok := p.getError(); ok {
				var zero T
				return zero, err
			}
		}
	}

	// Wait for completion
	p.mu.Lock()
	done := p.done
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-done:
	}

	// Get final result
	state := p.getState()
	if state == Fulfilled {
		if value, ok := p.getValue(); ok {
			return value, nil
		}
		var zero T
		return zero, &PromiseError{
			Message: "Promise fulfilled but value not available",
			Type:    RejectionError,
		}
	}

	var zero T
	if err, ok := p.getError(); ok {
		return zero, err
	}
	return zero, &PromiseError{
		Message: "Promise rejected but error not available",
		Type:    RejectionError,
	}
}

func (p *Promise[T]) State() State {
	return p.getState()
}

func (p *Promise[T]) IsPending() bool {
	return p.State() == Pending
}

func (p *Promise[T]) IsFulfilled() bool {
	return p.State() == Fulfilled
}

func (p *Promise[T]) IsRejected() bool {
	return p.State() == Rejected
}

// createPanicError creates a PromiseError from a panic value
func createPanicError(r interface{}, message string) error {
	if err, ok := r.(error); ok {
		return wrapErrorIfNeeded(err, message, PanicError)
	}
	return &PromiseError{
		Message: fmt.Sprintf("%s: %v", message, r),
		Cause:   nil,
		Type:    PanicError,
		Value:   r,
	}
}

func safeCallback[T any](callback func(T) any, value T, next *Promise[any]) {
	if next == nil {
		callback(value)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			next.reject(createPanicError(r, "panic in fulfilled callback"))
		}
	}()

	result := callback(value)
	next.resolve(result)
}

func safeErrorCallback(callback func(error) any, err error, next *Promise[any]) {
	if next == nil {
		callback(err)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := createPanicError(r, "panic in error callback")
			if promiseErr, ok := panicErr.(*PromiseError); ok {
				promiseErr.OriginalError = err
			}
			next.reject(panicErr)
		}
	}()

	result := callback(err)
	next.resolve(result)
}

func safeFinallyCallback[T any](callback func(), next *Promise[T], value T, err error) {
	if next == nil {
		callback()
		return
	}

	defer func() {
		if r := recover(); r != nil {
			panicErr := createPanicError(r, "panic in finally callback")
			if promiseErr, ok := panicErr.(*PromiseError); ok {
				promiseErr.OriginalError = err
			}
			next.reject(panicErr)
			return
		}

		if err == nil {
			next.resolve(value)
		} else {
			next.reject(err)
		}
	}()

	callback()
}
