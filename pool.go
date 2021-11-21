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

const (
	defaultIdleTimeout    = 2 * time.Second
	defaultFreeWorkerTime = 5 * time.Second
)

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
	link
	cond                    sync.Cond
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
		link:   link{},
		closed: make(chan struct{}),
		pool:   sync.Pool{},
	}
	p.cond = *sync.NewCond(&sync.Mutex{})
	p.pool.New = func() interface{} {
		return &worker{}
	}
	if p.capacity == 0 {
		p.capacity = math.MaxUint32
	}
	if p.idleTimeout == 0 {
		p.idleTimeout = defaultIdleTimeout
	}

	p.periodicalCleanTimeout()
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
	if p.IsClosed() {
		return ErrPoolIsClosed
	}

	p.cond.L.Lock()
	defer p.cond.L.Unlock()
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
		}
		p.cond.Wait()
		_ = atomic.AddUint32(&p.blockingNum, negativeOne)
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

func (p *Pool) getWorkerCache() *worker {
	return p.getHead(func(w *worker) bool {
		if w == nil || w.isClosed() {
			return false
		}
		return true
	})
}

func (p *Pool) createWorker() *worker {
	workingNum := atomic.LoadUint32(&p.workingNum)
	if workingNum < p.capacity {
		_ = atomic.AddUint32(&p.workingNum, 1)
		w := p.pool.Get().(*worker)
		w.start(p)
		return w
	}
	return nil
}

func (p *Pool) addBlockingNum() bool {
	if p.maxWaitTaskSize == 0 {
		return false
	}
	blockingNum := atomic.LoadUint32(&p.blockingNum)
	if blockingNum >= p.maxWaitTaskSize {
		return false
	}
	_ = atomic.AddUint32(&p.blockingNum, 1)
	return true
}

func (p *Pool) periodicalCleanTimeout() {
	tick := time.NewTicker(defaultFreeWorkerTime)
	go func() {
		for range tick.C {
			if p.IsClosed() {
				return
			}
			p.cond.L.Lock()
			p.deleteHead(func(w *worker) bool {
				if w == nil || w.isClosed() {
					return false
				}
				return true
			})
			p.cond.L.Unlock()
			p.cond.Broadcast()
		}
	}()
}
