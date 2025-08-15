package promise

import "runtime"

// MicrotaskConfig holds configuration for the microtask queue
type MicrotaskConfig struct {
	BufferSize  int
	WorkerCount int
}

// DefaultMicrotaskConfig returns the default configuration
func DefaultMicrotaskConfig() *MicrotaskConfig {
	return &MicrotaskConfig{
		BufferSize:  10000,
		WorkerCount: runtime.NumCPU() * 2,
	}
}

// microTask represents a microtask in the queue
type microTask struct {
	fn func()
}

// microTaskQueue manages microtasks with basic control
type microTaskQueue struct {
	tasks  chan *microTask
	config *MicrotaskConfig
}

// newMicrotaskQueue creates a new microtask queue
func newMicrotaskQueue(config *MicrotaskConfig) *microTaskQueue {
	queue := &microTaskQueue{
		tasks:  make(chan *microTask, config.BufferSize),
		config: config,
	}

	for i := 0; i < config.WorkerCount; i++ {
		go queue.worker()
	}

	return queue
}

// worker processes microtasks
func (q *microTaskQueue) worker() {
	for task := range q.tasks {
		if task != nil {
			task.fn()
		}
	}
}

// schedule schedules a microtask with basic control
func (q *microTaskQueue) schedule(fn func()) {
	task := &microTask{fn: fn}

	select {
	case q.tasks <- task:
		// Successfully queued
	default:
		// Queue full, execute directly to avoid deadlock
		go fn()
	}
}
