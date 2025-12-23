package promise

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
)

type taskPoolConfig struct {
	Workers   int
	QueueSize int
}

func defaultTaskPoolConfig() *taskPoolConfig {
	workers := runtime.NumCPU() * 2
	return &taskPoolConfig{
		Workers:   workers,
		QueueSize: workers * 2,
	}
}

type taskPool struct {
	tasks        chan func()
	workers      int32
	mu           sync.RWMutex
	shutdown     int32
	workerCtx    context.Context
	workerCancel context.CancelFunc
	workerWg     sync.WaitGroup
}

func newTaskPool(config *taskPoolConfig) *taskPool {
	if config == nil {
		config = defaultTaskPoolConfig()
	}

	if config.Workers <= 0 {
		config.Workers = runtime.NumCPU()
	}

	if config.QueueSize <= 0 {
		config.QueueSize = config.Workers * 2
	}

	ctx, cancel := context.WithCancel(context.Background())
	pool := &taskPool{
		tasks:        make(chan func(), config.QueueSize),
		workers:      int32(config.Workers),
		workerCtx:    ctx,
		workerCancel: cancel,
	}

	pool.startWorkers(config.Workers)
	return pool
}

func (p *taskPool) startWorkers(count int) {
	for i := 0; i < count; i++ {
		p.workerWg.Add(1)
		go p.worker()
	}
}

func (p *taskPool) worker() {
	defer p.workerWg.Done()

	for {
		select {
		case fn, ok := <-p.tasks:
			if !ok {
				return
			}
			if fn == nil {
				continue
			}

			// Execute task with panic recovery
			func() {
				defer func() {
					if r := recover(); r != nil {
						// Prevent worker crash
					}
				}()
				fn()
			}()

		case <-p.workerCtx.Done():
			return
		}
	}
}

func (p *taskPool) Submit(executor func()) error {
	if atomic.LoadInt32(&p.shutdown) == 1 {
		return ErrManagerStopped
	}

	select {
	case p.tasks <- executor:
		if atomic.LoadInt32(&p.shutdown) == 1 {
			return ErrManagerStopped
		}
		return nil
	case <-p.workerCtx.Done():
		return ErrManagerStopped
	}
}

func (p *taskPool) Workers() int {
	return int(atomic.LoadInt32(&p.workers))
}

func (p *taskPool) IsShutdown() bool {
	return atomic.LoadInt32(&p.shutdown) == 1
}

func (p *taskPool) Close() {
	if !atomic.CompareAndSwapInt32(&p.shutdown, 0, 1) {
		return
	}

	p.workerCancel()

	p.mu.Lock()
	close(p.tasks)
	p.mu.Unlock()

	p.workerWg.Wait()
}

func (p *taskPool) WaitForShutdown() {
	p.workerWg.Wait()
}
