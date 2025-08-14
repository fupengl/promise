package promise

import "runtime"

// MicrotaskConfig holds basic configuration for the microtask queue
type MicrotaskConfig struct {
	// BufferSize is the size of the task buffer
	BufferSize int
	// WorkerCount is the number of worker goroutines
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
	fn   func()
	done chan struct{}
}

// microTaskQueue manages microtasks
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

	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		go queue.worker()
	}

	return queue
}

// worker processes microtasks
func (q *microTaskQueue) worker() {
	for task := range q.tasks {
		task.fn()
		close(task.done)
	}
}
