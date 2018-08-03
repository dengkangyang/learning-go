package ants

import (
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type sig struct{}

type f func() error

type Pool struct {
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
	workers []*Worker

	// release is used to notice the pool to closed itself.
	release chan sig

	// lock for synchronous operation
	lock sync.Mutex

	once sync.Once
}

func (p *Pool) periodicallyPurge() {
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
			if currentTime.Sub(w.recyleTime) <= p.expiryDuration {
				break
			}
			n = i
			<-p.freeSignal
			w.task <- nil
			idleWorkers[i] = nil // 将索引为 n的worker从idleworkers中删除
		}
		if n > 0 {
			n++ // 将索引为 n的worker从idleworkers中删除
			p.workers = idleWorkers[n:]
		}
		p.lock.Unlock()
	}
}

// NewPool generates a instance of ants pool
func NewPool(size int) (*Pool, error) {
	return NewTimingPool(size, DefaultCleanIntervalTime)
}

// NewTimingPool generates a instance of ants pool with a custom timed task
func NewTimingPool(size, expiry int) (*Pool, error) {
	if siez <= 0 {
		return nil, ErrInvalidPoolSize
	}
	if expiry <= 0 {
		return nil, ErrInvalidPoolExpiry
	}

	p := &Pool{
		capacity:       int32(size),
		freeSignal:     make(chan sig, math.MaxInt32),
		release:        make(chan sig, 1),
		expiryDuration: time.Duration(expiry) * time.Second,
	}
	go p.periodicallyPurge()
	return p, nil
}

func (p *Pool) Submit(task f) error {
	if len(p.release) > 0 {
		return ErrPoolClosed
	}
	w := p.getWorker()
	w.task <- task
	return nil
}

// Running returns the number of the currently running goroutines
func (p *Pool) Running() int {
	return int(atomic.LoadInt32(&p.running))
}

// Free returns the available goroutines to work
func (p *Pool) Free() int {
	return int(atomic.LoadInt32(&p.capacity) - atomic.LoadInt32(&p.running))
}

// Cap returns the capacity of this pool
func (p *Pool) Cap() int {
	return int(atomic.LoadInt32(&p.capacity))
}

// ReSize change the capacity of this pool
func (p *Pool) ReSize(size int) {
	if size == p.Cap() {
		return
	}
	atomic.StoreInt32(&p.capacity, int32(size))
	diff := p.Running() - size
	if diff > 0 {
		for i := 0; i < diff; i++ {
			p.getWorker().task <- nil
		}
	}
}

// Release Closed this pool
func (p *Pool) Release() error {
	p.once.Do(func() { // 保证只释放一次
		p.release <- sig{}
		p.lock.Lock()
		idleWorkers := p.workers
		for i, w := range idleWorkers {
			<-p.freeSignal
			w.task <- nil
			idleWorkers[i] = nil
		}
		p.workers = nil
		p.lock.Unlock()
	})
	return nil
}

// incrRunning increases the number of the currently running goroutines
func (p *Pool) incrRunning() {
	atomic.AddInt32(&p.running, 1)
}

// decrRunning decreases the number of the currently running goroutines
func (p *Pool) decrRunning() {
	atomic.AddInt32(&p.running, -1)
}

// getWorker returns a available worker to run the tasks.
func (p *Pool) getWorker() *Worker {
	var w *Worker
	waiting := false

	p.lock.Lock()
	idleWorkers := p.workers
	n := len(idleWorkers) - 1
	if n < 0 { // 说明 pool中没有worker了
		waiting = p.Running() >= p.Cap()
	} else { // 说明pool中有worker
		<-p.freeSignal     // 等待 putWorker操作，这一步还是加了这个等待信号，让putWorker来触发一次取w操作？
		w = idleWorkers[n] // 从末尾中取一个w
		idleWorkers[n] = nil
		p.workers = idleWorkers[:n]
	}
	p.lock.Unlock()

	if waiting {
		<-p.freeSignal // 等待 putWorker操作
		p.lock.Lock()
		idleWorkers = p.workers
		l := len(idleWorkers) - 1
		w = idleWorkers[l]
		idleWorkers[l] = nil
		p.workers = idleWorkers[:l]
		p.lock.Unlock()
	} else if w == nil {
		w = &Worker{
			pool: p,
			task: make(chan f, 1),
		}
		w.run()
		p.incrRunning()
	}
	return w
}

func (p *Pool) putWorker(worker *Worker) {
	worker.recycleTime = time.Now()
	p.lock.Lock()
	p.workers = append(p.workers, worker)
	p.lock.Unlock()
	p.freeSignal <- sig{}
}
