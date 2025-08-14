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
		BufferSize:  1000,
		WorkerCount: runtime.NumCPU(),
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

var globalMicroTaskQueue = newMicrotaskQueue(DefaultMicrotaskConfig())

// SetMicrotaskConfig sets the configuration for the microtask queue
// This must be called before any promises are created
func SetMicrotaskConfig(config *MicrotaskConfig) {
	if config == nil {
		config = DefaultMicrotaskConfig()
	}
	globalMicroTaskQueue = newMicrotaskQueue(config)
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

// scheduleMicrotask schedules a function to run as a microtask
func scheduleMicrotask(fn func()) {
	// Create a new channel for each task
	done := make(chan struct{})

	task := &microTask{
		fn:   fn,
		done: done,
	}

	// Always queue the task, even if it means blocking
	// This preserves the microtask execution order
	globalMicroTaskQueue.tasks <- task
}
