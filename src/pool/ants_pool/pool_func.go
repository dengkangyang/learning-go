package ants

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type pf func(interface{}) error

// PoolWithFunc accept the tasks from client,it limits the total
// of goroutines to a given number by recycling goroutines.
type PoolWithFunc struct {
	// capacity of the pool.
	capacity int32

	// running is the number of the currently running goroutines.
	running int32

	// expiryDuration set the expired time (second) of every worker.
	expiryDuration time.Duration

	// freeSignal is used to notice pool there are available
	// workers which can be sent to work.
	freeSignal chan sig

	// workers is a slice that store the available workers.
	workers []*WorkerWithFunc

	// release is used to notice the pool to closed itself.
	release chan sig

	// lock for synchronous operation
	lock sync.Mutex

	// pf is the function for processing tasks
	poolFunc pf

	once sync.Once
}

func (p *PoolWithFunc) periodicallyPurge() {
	heartbeat := time.NewTicker(p.expiryDuration)
	for range heartbeat.C {
		currentTime := time.Now()
		p.lock.Lock()
		idleWorkers := p.workers
		if len(idleWorkers) == 0 && p.Running() == 0 && len(p.release) > 0 {
			p.lock.Unlock()
			return
		}
		n := 0
		for i, w := range idleWorkers {
			if currentTime.Sub(w.recycleTime) <= p.expiryDuration {
				break
			}
			n = i
			<-p.freeSignal
			w.args <- nil
			idleWorkers[i] = nil
		}
		if n > 0 {
			n++
			p.workers = idleWorkers[n:]
		}
		p.lock.Unlock()
	}
}

// NewPoolWithFunc generates a instance of ants pool with a specific function
func NewPoolWithFunc(size int, f pf) (*PoolWithFunc, error) {
	return NewTimingPoolWithFunc(size, DefaultCleanIntervalTime, f)
}

// NewTimingPoolWithFunc generates a instance of ants pool with a specific function and a custom timed task
func NewTimingPoolWithFunc(size, expiry int, f pf) (*PoolWithFunc, error) {
	if size <= 0 {
		return nil, ErrInvalidPoolSize
	}
	if expiry <= 0 {
		return nil, ErrInvalidPoolExpiry
	}
	p := &PoolWithFunc{
		capacity:       int32(size),
		freeSignal:     make(chan sig, math.MaxInt32),
		release:        make(chan sig, 1),
		expiryDuration: time.Duration(expiry) * time.Second,
		poolFunc:       f,
	}
	go p.periodicallyPurge()
	return p, nil
}

//-------------------------------------------------------------------------

// Serve submit a task to pool
func (p *PoolWithFunc) Serve(args interface{}) error {
	//if atomic.LoadInt32(&p.closed) == 1 {
	//	return ErrPoolClosed
	//}
	if len(p.release) > 0 {
		return ErrPoolClosed
	}
	w := p.getWorker()
	w.args <- args
	return nil
}

// Running returns the number of the currently running goroutines
func (p *PoolWithFunc) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free returns the available goroutines to work
func (p *PoolWithFunc) Free() int {
	return int(atomic.LoadInt32(&p.capacity) - atomic.LoadInt32(&p.running))
}

// Cap returns the capacity of this pool
func (p *PoolWithFunc) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// ReSize change the capacity of this pool
func (p *PoolWithFunc) ReSize(size int) {
	if size == p.Cap() {
		return
	}
	atomic.StoreInt32(&p.capacity, int32(size))
	diff := p.Running() - size
	if diff > 0 {
		for i := 0; i < diff; i++ {
			p.getWorker().args <- nil
		}
	}
}

// Release Closed this pool
func (p *PoolWithFunc) Release() error {
	p.once.Do(func() {
		p.release <- sig{}
		p.lock.Lock()
		idleWorkers := p.workers
		for i, w := range idleWorkers {
			<-p.freeSignal
			w.args <- nil
			idleWorkers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
	})
	return nil
}

//-------------------------------------------------------------------------

// incrRunning increases the number of the currently running goroutines
func (p *PoolWithFunc) incrRunning() {
	atomic.AddInt32(&p.running, 1)
}

// decrRunning decreases the number of the currently running goroutines
func (p *PoolWithFunc) decrRunning() {
	atomic.AddInt32(&p.running, -1)
}

// getWorker returns a available worker to run the tasks.
func (p *PoolWithFunc) getWorker() *WorkerWithFunc {
	var w *WorkerWithFunc
	waiting := false

	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 {
		waiting = p.Running() >= p.Cap()
	} else {
		<-p.freeSignal
		w = idleWorkers[n]
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
	}
	p.lock.Unlock()

	if waiting {
		<-p.freeSignal
		p.lock.Lock()
		idleWorkers = p.workers
		l := len(idleWorkers) - 1
		w = idleWorkers[l]
		idleWorkers[l] = nil
		p.workers = idleWorkers[:l]
		p.lock.Unlock()
	} else if w == nil {
		w = &WorkerWithFunc{
			pool: p,
			args: make(chan interface{}, 1),
		}
		w.run()
		p.incrRunning()
	}
	return w
}

// putWorker puts a worker back into free pool, recycling the goroutines.
func (p *PoolWithFunc) putWorker(worker *WorkerWithFunc) {
	worker.recycleTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, worker)
	p.lock.Unlock()
	p.freeSignal <- sig{}
}
