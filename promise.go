package promise

import (
	"context"
	"errors"
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
			Cause:   errors.New("manager is shutdown"),
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
			Cause:   errors.New("manager is shutdown"),
			Type:    RejectionError,
		})
		return p
	}

	if err := manager.scheduleExecutor(func() {
		defer func() {
			if r := recover(); r != nil {
				var panicErr error
				if err, ok := r.(error); ok {
					panicErr = wrapErrorIfNeeded(err, "panic in executor", PanicError)
				} else {
					panicErr = &PromiseError{
						Message: fmt.Sprintf("panic in executor: %v", r),
						Cause:   nil,
						Type:    PanicError,
						Value:   r,
					}
				}
				p.reject(panicErr)
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
	select {
	case <-p.done:
		p.mu.Unlock()
		return
	default:
		close(p.done)
	}

	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

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

func (p *Promise[T]) reject(err error) {
	if !p.setState(Rejected) {
		return
	}

	finalErr := wrapErrorIfNeeded(err, "Promise rejected", RejectionError)
	p.setError(finalErr)

	p.mu.Lock()
	select {
	case <-p.done:
		p.mu.Unlock()
		return
	default:
		close(p.done)
	}

	handlers := p.handlers
	p.handlers = nil
	p.mu.Unlock()

	for _, h := range handlers {
		if h.onRejected != nil {
			handler := h.onRejected
			next := h.next
			errVal := finalErr

			p.manager.scheduleMicrotask(func() {
				safeErrorCallback(handler, errVal, next)
			})
		} else if h.next != nil {
			next := h.next
			errVal := finalErr

			p.manager.scheduleMicrotask(func() {
				next.reject(errVal)
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

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.getState() == Fulfilled {
		handler := h.onFulfilled
		val, _ := p.getValue()

		p.manager.scheduleMicrotask(func() {
			handler(val)
		})
	} else if p.getState() == Rejected {
		handler := h.onRejected
		errVal, _ := p.getError()

		p.manager.scheduleMicrotask(func() {
			handler(errVal)
		})
	} else {
		if p.handlers == nil {
			p.handlers = make([]*handler[T], 0, 2)
		}
		p.handlers = append(p.handlers, h)
	}

	return next
}

func (p *Promise[T]) Await() (T, error) {
	state := p.getState()
	if state == Fulfilled {
		if value, ok := p.getValue(); ok {
			return value, nil
		}
	}
	if state == Rejected {
		if err, ok := p.getError(); ok {
			var zero T
			return zero, err
		}
	}

	p.mu.Lock()
	done := p.done
	p.mu.Unlock()

	<-done

	state = p.getState()
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

func (p *Promise[T]) AwaitWithContext(ctx context.Context) (T, error) {
	state := p.getState()
	if state == Fulfilled {
		if value, ok := p.getValue(); ok {
			return value, nil
		}
	}
	if state == Rejected {
		if err, ok := p.getError(); ok {
			var zero T
			return zero, err
		}
	}

	p.mu.Lock()
	done := p.done
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case <-done:
	}

	state = p.getState()
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

func safeCallback[T any](callback func(T) any, value T, next *Promise[any]) {
	if next == nil {
		callback(value)
		return
	}

	defer func() {
		if r := recover(); r != nil {
			var err error
			if e, ok := r.(error); ok {
				err = &PromiseError{
					Message: "panic in fulfilled callback",
					Cause:   e,
					Type:    PanicError,
				}
			} else {
				err = &PromiseError{
					Message: fmt.Sprintf("panic in fulfilled callback: %v", r),
					Cause:   nil,
					Type:    PanicError,
					Value:   r,
				}
			}
			next.reject(err)
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
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = &PromiseError{
					Message:       "panic in error callback",
					Cause:         e,
					Type:          PanicError,
					OriginalError: err,
				}
			} else {
				panicErr = &PromiseError{
					Message:       fmt.Sprintf("panic in error callback: %v", r),
					Cause:         nil,
					Type:          PanicError,
					Value:         r,
					OriginalError: err,
				}
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
			var panicErr error
			if e, ok := r.(error); ok {
				panicErr = &PromiseError{
					Message:       "panic in finally callback",
					Cause:         e,
					Type:          PanicError,
					OriginalError: err,
				}
			} else {
				panicErr = &PromiseError{
					Message:       fmt.Sprintf("panic in finally callback: %v", r),
					Cause:         nil,
					Type:          PanicError,
					Value:         r,
					OriginalError: err,
				}
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
