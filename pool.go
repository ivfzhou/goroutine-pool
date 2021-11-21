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

var taskCache = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

// Pool 协程池结构体。
type Pool struct {
	*option
	workingNum      uint32
	blockingNum     uint32
	closed          chan struct{}
	workerCache     *Link
	workingNumLock  sync.Mutex
	blockingNumLock sync.Mutex
}

// New 新建协程池。
//
// opts 协程池参数。
func New(opts ...optionFunc) *Pool {
	p := &Pool{
		option:      loadOption(opts...),
		workerCache: &Link{},
		closed:      make(chan struct{}),
	}
	if p.capacity == 0 {
		p.capacity = math.MaxUint32
	}
	if p.idleTimeout == 0 {
		p.idleTimeout = defaultIdleTimeout
	}
	go func() {
		tick := time.NewTicker(defaultFreeWorkerTime)
		for range tick.C {
			if p.IsClosed() {
				return
			}
			p.workerCache.CheckHead(func(v interface{}) bool {
				if w, ok := v.(*worker); !ok || w.isClosed() {
					return false
				}
				return true
			})
		}
	}()
	return p
}

// Submit 提交任务。
//
// ctx 该任务上下文。
//
// fn 任务函数。
func (p *Pool) Submit(ctx context.Context, fn func(context.Context)) error {
	isBlocked := false
	defer func() {
		if isBlocked {
			_ = atomic.AddUint32(&p.blockingNum, ^uint32(1)+1)
		}
	}()
	for {
		select {
		case <-p.closed:
			return ErrPoolIsClosed
		default:
		}

		w, _ := p.workerCache.GetHead(func(v interface{}) bool {
			if w, ok := v.(*worker); !ok || w.isClosed() {
				return false
			}
			return true
		}).(*worker)
		if w != nil {
			w.work(ctx, fn)
			return nil
		}

		w = p.atomicallyAddWorkingNum()
		if w != nil {
			w.work(ctx, fn)
			return nil
		}
		if p.maxWaitTaskSize == 0 {
			return ErrPoolIsOverload
		} else if p.atomicallyAddBlockNum() {
			isBlocked = true
		} else {
			return ErrPoolIsOverload
		}
		runtime.Gosched()
	}
}

// Close 关闭协程池。
//
// error 关闭失败返回错误。
func (p *Pool) Close() error {
	close(p.closed)
	return nil
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

func (p *Pool) atomicallyAddWorkingNum() *worker {
	p.workingNumLock.Lock()
	workingNum := atomic.LoadUint32(&p.workingNum)
	if workingNum < p.capacity {
		_ = atomic.AddUint32(&p.workingNum, 1)
		p.workingNumLock.Unlock()
		w := &worker{
			Pool: p,
		}
		w.start()
		return w
	}
	p.workingNumLock.Unlock()
	return nil
}

func (p *Pool) atomicallyAddBlockNum() bool {
	p.blockingNumLock.Lock()
	blockingNum := atomic.LoadUint32(&p.blockingNum)
	if blockingNum < p.maxWaitTaskSize {
		_ = atomic.AddUint32(&p.blockingNum, 1)
		p.blockingNumLock.Unlock()
		return true
	}
	p.blockingNumLock.Unlock()
	return false
}
