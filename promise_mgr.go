package promise

import (
	"errors"
	"runtime"
	"sync"
)

// PromiseMgr manages both executor and microtask execution
type PromiseMgr struct {
	tasks    chan *executorTask
	workers  int
	mu       sync.RWMutex
	shutdown bool

	// Microtask queue management
	microtaskQueue  *microTaskQueue
	microtaskConfig *MicrotaskConfig
}

// executorTask represents a task to be executed by the manager
type executorTask struct {
	executor func()
	done     chan struct{}
}

// NewPromiseMgr creates a new Promise manager
func NewPromiseMgr(workers int) *PromiseMgr {
	if workers <= 0 {
		workers = runtime.NumCPU() * 2
	}
	return NewPromiseMgrWithConfig(workers, DefaultMicrotaskConfig())
}

// NewPromiseMgrWithConfig creates a new Promise manager with custom microtask configuration
func NewPromiseMgrWithConfig(workers int, microtaskConfig *MicrotaskConfig) *PromiseMgr {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	if microtaskConfig == nil {
		microtaskConfig = DefaultMicrotaskConfig()
	}

	manager := &PromiseMgr{
		tasks:           make(chan *executorTask, workers*2),
		workers:         workers,
		microtaskConfig: microtaskConfig,
		microtaskQueue:  newMicrotaskQueue(microtaskConfig),
	}

	// Start worker goroutines
	for i := 0; i < workers; i++ {
		go manager.worker()
	}

	return manager
}

// worker processes tasks from the manager
func (m *PromiseMgr) worker() {
	for task := range m.tasks {
		if task == nil {
			return // shutdown signal
		}

		func() {
			task.executor()
			close(task.done)
		}()
	}
}

// scheduleExecutor schedules a task to the manager
func (m *PromiseMgr) scheduleExecutor(executor func()) {
	task := &executorTask{
		executor: executor,
		done:     make(chan struct{}),
	}

	select {
	case m.tasks <- task:
		// Task submitted successfully
	default:
		// Execute immediately if channel is full
		executor()
	}
}

// ScheduleMicrotask schedules a microtask using this manager's queue
func (m *PromiseMgr) scheduleMicrotask(fn func()) {
	done := make(chan struct{})
	task := &microTask{
		fn:   fn,
		done: done,
	}

	// Submit to queue and wait for completion
	m.microtaskQueue.tasks <- task
	<-done
}

// GetMicrotaskConfig returns the current microtask configuration
func (m *PromiseMgr) GetMicrotaskConfig() *MicrotaskConfig {
	return m.microtaskConfig
}

// SetMicrotaskConfig updates the microtask configuration
// This should be called before any promises are created with this manager
func (m *PromiseMgr) SetMicrotaskConfig(config *MicrotaskConfig) error {
	if config == nil {
		config = DefaultMicrotaskConfig()
	}

	// Check if there are running microtasks
	if len(m.microtaskQueue.tasks) > 0 {
		return errors.New("cannot update config while microtasks are running")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.microtaskConfig = config
	m.microtaskQueue = newMicrotaskQueue(config)

	return nil
}

// SetExecutorWorker updates the number of executor worker goroutines
// This should be called before any promises are created with this manager
func (m *PromiseMgr) SetExecutorWorker(workers int) error {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	// Check if there are running tasks
	if len(m.tasks) > 0 {
		return errors.New("cannot update workers while tasks are running")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.workers = workers
	// Recreate task channel
	m.tasks = make(chan *executorTask, workers*2)

	// Restart worker goroutines
	for i := 0; i < workers; i++ {
		go m.worker()
	}

	return nil
}

// Close gracefully shuts down the manager
func (m *PromiseMgr) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.shutdown {
		m.shutdown = true
		close(m.tasks)
	}
}

// Workers returns the number of worker goroutines
func (m *PromiseMgr) Workers() int {
	return m.workers
}

// IsShutdown returns true if the manager is shut down
func (m *PromiseMgr) IsShutdown() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.shutdown
}

// global default manager instance
var defaultManager *PromiseMgr
var defaultManagerOnce sync.Once

// GetDefaultMgr returns the default Promise manager
func GetDefaultMgr() *PromiseMgr {
	defaultManagerOnce.Do(func() {
		defaultManager = NewPromiseMgr(runtime.NumCPU())
	})
	return defaultManager
}

// ResetDefaultMgr resets the default manager with new configuration
// This should be called before any promises are created
func ResetDefaultMgr(workers int, microtaskConfig *MicrotaskConfig) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	if microtaskConfig == nil {
		microtaskConfig = DefaultMicrotaskConfig()
	}

	// Reset default manager
	defaultManager = NewPromiseMgrWithConfig(workers, microtaskConfig)
}
