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

var taskObjectPool = sync.Pool{
	New: func() interface{} {
		return &task{
			Done: make(chan struct{}),
		}
	},
}

func getTaskFromPool() *task {
	t := taskObjectPool.Get().(*task)
	t.Executor = nil
	t.Done = make(chan struct{})
	return t
}

func putTaskToPool(t *task) {
	t.Executor = nil
	t.Done = nil
	taskObjectPool.Put(t)
}

func defaultTaskPoolConfig() *taskPoolConfig {
	workers := runtime.NumCPU() * 2
	return &taskPoolConfig{
		Workers:   workers,
		QueueSize: workers * 2,
	}
}

type task struct {
	Executor func()
	Done     chan struct{}
}

type taskPool struct {
	tasks        chan *task
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
		tasks:        make(chan *task, config.QueueSize),
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
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			if task == nil {
				continue
			}

			func() {
				defer func() {
					if r := recover(); r != nil {
						// Prevent worker crash
					}
				}()
				task.Executor()
			}()

			if task.Done != nil {
				select {
				case <-task.Done:
					// Channel already closed, do nothing
				default:
					close(task.Done)
				}
			}
			putTaskToPool(task)

		case <-p.workerCtx.Done():
			return
		}
	}
}

func (p *taskPool) Submit(executor func()) error {
	if atomic.LoadInt32(&p.shutdown) == 1 {
		return ErrManagerStopped
	}

	task := getTaskFromPool()
	task.Executor = executor

	select {
	case p.tasks <- task:
		return nil
	default:
		putTaskToPool(task)
		go func() {
			defer func() {
				if r := recover(); r != nil {
					// Handle panic
				}
			}()
			executor()
		}()
		return nil
	}
}

func (p *taskPool) SubmitAndWait(executor func()) error {
	if atomic.LoadInt32(&p.shutdown) == 1 {
		return ErrManagerStopped
	}

	task := getTaskFromPool()
	task.Executor = executor

	select {
	case p.tasks <- task:
		<-task.Done
		putTaskToPool(task)
		return nil
	default:
		putTaskToPool(task)
		executor()
		return nil
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

	p.mu.Lock()
	defer p.mu.Unlock()

	p.workerCancel()
	close(p.tasks)
	p.workerWg.Wait()
}

func (p *taskPool) WaitForShutdown() {
	p.workerWg.Wait()
}
