// Package goroutine_pool 高性能的协程管理池。
package goroutine_pool

import (
	"context"
	"errors"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

const defaultIdleTimeout = 2 * time.Second

var (
	// ErrPoolIsOverload 代表池已满。
	ErrPoolIsOverload = errors.New("goroutine pool is overload")
	// ErrPoolIsClosed 代表池已关闭。
	ErrPoolIsClosed = errors.New("goroutine pool is closed")
)

var (
	taskCache = func() int {
		if runtime.GOMAXPROCS(0) == 1 {
			return 0
		}
		return 1
	}()
	negativeOne = ^uint32(1) + 1
)

// Pool 协程池结构体。
type Pool struct {
	option
	cache                   chan *worker
	lock                    sync.Mutex
	closed                  chan struct{}
	pool                    sync.Pool
	workingNum, blockingNum uint32
}

// New 新建协程池。
//
// opts 协程池参数。
func New(opts ...optionFunc) *Pool {
	p := &Pool{
		option: loadOption(opts...),
		closed: make(chan struct{}),
		pool:   sync.Pool{},
	}
	if p.capacity == 0 {
		p.capacity = math.MaxUint32
	}
	if p.idleTimeout == 0 {
		p.idleTimeout = defaultIdleTimeout
	}

	p.lock = sync.Mutex{}
	p.pool.New = func() interface{} {
		return &worker{}
	}
	p.cache = make(chan *worker, p.capacity/3)
	return p
}

// Submit 提交任务。
//
// ctx 该任务上下文。
//
// fn 任务函数。
//
// error 异常返回。
func (p *Pool) Submit(ctx context.Context, fn func(context.Context)) error {
	isBlock := false
	defer func() {
		if isBlock {
			_ = atomic.AddUint32(&p.blockingNum, negativeOne)
		}
	}()
	for {
		if p.IsClosed() {
			return ErrPoolIsClosed
		}

		w := p.getWorkerCache()
		if w != nil {
			w.work(ctx, fn)
			return nil
		}

		w = p.createWorker()
		if w != nil {
			w.work(ctx, fn)
			return nil
		}

		if !p.addBlockingNum() {
			return ErrPoolIsOverload
		} else {
			isBlock = true
			runtime.Gosched()
		}
	}
}

// Close 关闭协程池。
//
// error 关闭失败返回错误。
func (p *Pool) Close() {
	close(p.closed)
}

// IsClosed 协程池是否已关闭。
//
// bool true 代表已关闭。false 代表未关闭。
func (p *Pool) IsClosed() bool {
	select {
	case <-p.closed:
		return true
	default:
		return false
	}
}

// Cap 最大协程数。
//
// uint32 最大协程数。
func (p *Pool) Cap() uint32 {
	return p.capacity
}

// CurrentBlockingSize 当前阻塞的任务数。
//
// uint32 当前阻塞的任务数。
func (p *Pool) CurrentBlockingSize() uint32 {
	return atomic.LoadUint32(&p.blockingNum)
}

// CurrentWorkingNum 当前运行的协程数。
//
// uint32 当前运行的协程数。
func (p *Pool) CurrentWorkingNum() uint32 {
	return atomic.LoadUint32(&p.workingNum)
}

func (p *Pool) getWorkerCache() (w *worker) {
	for len(p.cache) != 0 {
		w = <-p.cache
		if w.isTimeout(p.idleTimeout) {
			w.close()
		} else {
			return
		}
	}
	return nil
}

func (p *Pool) createWorker() *worker {
	p.lock.Lock()
	workingNum := atomic.LoadUint32(&p.workingNum)
	if workingNum < p.capacity {
		_ = atomic.AddUint32(&p.workingNum, 1)
		p.lock.Unlock()
		w := p.pool.Get().(*worker)
		w.start(p)
		return w
	}
	p.lock.Unlock()
	return nil
}

func (p *Pool) addBlockingNum() bool {
	if p.maxWaitTaskSize == 0 {
		return false
	}
	p.lock.Lock()
	defer p.lock.Unlock()
	blockingNum := atomic.LoadUint32(&p.blockingNum)
	if blockingNum < p.maxWaitTaskSize {
		_ = atomic.AddUint32(&p.blockingNum, 1)
		return true
	}
	return false
}
