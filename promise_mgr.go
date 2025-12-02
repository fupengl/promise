package promise

import (
	"runtime"
	"sync"
	"sync/atomic"
)

type PromiseMgrConfig struct {
	ExecutorWorkers    int // Executor worker count
	ExecutorQueueSize  int // Executor queue size
	MicrotaskWorkers   int // Microtask worker count
	MicrotaskQueueSize int // Microtask queue size
}

func DefaultPromiseMgrConfig() *PromiseMgrConfig {
	cpuCount := runtime.NumCPU()
	executorWorkers := cpuCount * 2
	microtaskWorkers := cpuCount

	return &PromiseMgrConfig{
		ExecutorWorkers:    executorWorkers,
		ExecutorQueueSize:  executorWorkers * 4,
		MicrotaskWorkers:   microtaskWorkers,
		MicrotaskQueueSize: microtaskWorkers * 100,
	}
}

type PromiseMgr struct {
	executorPool  *taskPool
	microtaskPool *taskPool
	mu            sync.RWMutex
	shutdown      int32
	config        *PromiseMgrConfig
}

func NewPromiseMgr() *PromiseMgr {
	return NewPromiseMgrWithConfig(DefaultPromiseMgrConfig())
}

func NewPromiseMgrWithConfig(config *PromiseMgrConfig) *PromiseMgr {
	if config == nil {
		config = DefaultPromiseMgrConfig()
	}

	defaultConfig := DefaultPromiseMgrConfig()
	normalizeConfig(config, defaultConfig)

	return &PromiseMgr{
		executorPool: newTaskPool(&taskPoolConfig{
			Workers:   config.ExecutorWorkers,
			QueueSize: config.ExecutorQueueSize,
		}),
		microtaskPool: newTaskPool(&taskPoolConfig{
			Workers:   config.MicrotaskWorkers,
			QueueSize: config.MicrotaskQueueSize,
		}),
		config: config,
	}
}

// normalizeConfig normalizes config values using defaults
func normalizeConfig(config *PromiseMgrConfig, defaultConfig *PromiseMgrConfig) {
	if config.ExecutorWorkers <= 0 {
		config.ExecutorWorkers = defaultConfig.ExecutorWorkers
	}
	if config.ExecutorQueueSize <= 0 {
		config.ExecutorQueueSize = defaultConfig.ExecutorQueueSize
	}
	if config.MicrotaskWorkers <= 0 {
		config.MicrotaskWorkers = defaultConfig.MicrotaskWorkers
	}
	if config.MicrotaskQueueSize <= 0 {
		config.MicrotaskQueueSize = defaultConfig.MicrotaskQueueSize
	}
}

// scheduleExecutor schedules a task to the manager
func (m *PromiseMgr) scheduleExecutor(executor func()) error {
	if atomic.LoadInt32(&m.shutdown) == 1 {
		return ErrManagerStopped
	}

	return m.executorPool.Submit(executor)
}

// scheduleMicrotask schedules a microtask using this manager's queue
func (m *PromiseMgr) scheduleMicrotask(fn func()) error {
	if atomic.LoadInt32(&m.shutdown) == 1 {
		return ErrManagerStopped
	}

	return m.microtaskPool.Submit(fn)
}

// SetMicrotaskConfig updates microtask configuration
func (m *PromiseMgr) SetMicrotaskConfig(workers int, queueSize int) *PromiseMgr {
	if atomic.LoadInt32(&m.shutdown) == 1 {
		return m
	}

	defaultConfig := DefaultPromiseMgrConfig()
	if workers <= 0 {
		workers = defaultConfig.MicrotaskWorkers
	}
	if queueSize <= 0 {
		queueSize = defaultConfig.MicrotaskQueueSize
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.microtaskPool != nil {
		m.microtaskPool.Close()
	}

	m.config.MicrotaskWorkers = workers
	m.config.MicrotaskQueueSize = queueSize
	m.microtaskPool = newTaskPool(&taskPoolConfig{
		Workers:   workers,
		QueueSize: queueSize,
	})

	return m
}

// SetExecutorConfig updates executor configuration
func (m *PromiseMgr) SetExecutorConfig(workers int, queueSize int) *PromiseMgr {
	if atomic.LoadInt32(&m.shutdown) == 1 {
		return m
	}

	defaultConfig := DefaultPromiseMgrConfig()
	if workers <= 0 {
		workers = defaultConfig.ExecutorWorkers
	}
	if queueSize <= 0 {
		queueSize = defaultConfig.ExecutorQueueSize
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.executorPool != nil {
		m.executorPool.Close()
	}

	m.config.ExecutorWorkers = workers
	m.config.ExecutorQueueSize = queueSize
	m.executorPool = newTaskPool(&taskPoolConfig{
		Workers:   workers,
		QueueSize: queueSize,
	})

	return m
}

// Close shuts down the manager
func (m *PromiseMgr) Close() {
	if !atomic.CompareAndSwapInt32(&m.shutdown, 0, 1) {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.executorPool.Close()
	m.microtaskPool.Close()
}

// GetConfig returns the current configuration
func (m *PromiseMgr) GetConfig() *PromiseMgrConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()

	config := *m.config
	return &config
}

// IsShutdown returns true if the manager is shut down
func (m *PromiseMgr) IsShutdown() bool {
	return atomic.LoadInt32(&m.shutdown) == 1
}

// WaitForShutdown waits for shutdown completion
func (m *PromiseMgr) WaitForShutdown() {
	m.executorPool.WaitForShutdown()
	m.microtaskPool.WaitForShutdown()
}

var defaultManager *PromiseMgr
var defaultManagerOnce sync.Once
var defaultManagerMu sync.RWMutex

// GetDefaultMgr returns the default Promise manager
func GetDefaultMgr() *PromiseMgr {
	defaultManagerOnce.Do(func() {
		defaultManager = NewPromiseMgr()
	})

	defaultManagerMu.RLock()
	defer defaultManagerMu.RUnlock()
	return defaultManager
}

// ResetDefaultMgrExecutor resets executor configuration of default manager
func ResetDefaultMgrExecutor(workers int, queueSize int) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()

	if defaultManager != nil && !defaultManager.IsShutdown() {
		defaultManager.SetExecutorConfig(workers, queueSize)
	}
}

// ResetDefaultMgrMicrotask resets microtask configuration of default manager
func ResetDefaultMgrMicrotask(workers int, queueSize int) {
	defaultManagerMu.Lock()
	defer defaultManagerMu.Unlock()

	if defaultManager != nil && !defaultManager.IsShutdown() {
		defaultManager.SetMicrotaskConfig(workers, queueSize)
	}
}
