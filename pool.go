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

	"gitee.com/ivfzhou/safe-queue"
)

const defaultIdleTimeout = 2 * time.Second

var (
	// ErrPollIsOverload 代表池已满。
	ErrPollIsOverload = errors.New("goroutine Pool is overload")
	// ErrPoolIsClosed 代表池已关闭。
	ErrPoolIsClosed = errors.New("goroutine Pool is closed")
)

var taskCache = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

type Pool struct {
	*option
	workingNum      uint32
	blockingNum     uint32
	closed          uint32
	workerPoll      sync.Pool
	workerCache     *safe_queue.Queue
	workingNumLock  sync.Mutex
	blockingNumLock sync.Mutex
	once            sync.Once
	init            uint32
}

// New 新建协程池。
func New(opts ...optionFunc) *Pool {
	p := &Pool{
		option: loadOption(opts...),
		init:   1,
	}
	if p.capacity == 0 {
		p.capacity = math.MaxUint32
	}
	if p.idleTimeout == 0 {
		p.idleTimeout = defaultIdleTimeout
	}
	p.workerPoll.New = func() interface{} { return &worker{Pool: p} }
	p.workerCache = safe_queue.New(p.capacity)
	return p
}

func (p *Pool) Submit(ctx context.Context, fn func(context.Context)) error {
	p.once.Do(func() {
		if p.init == 0 {
			*p = *New()
		}
	})
	for {
		if atomic.LoadUint32(&p.closed) != 0 {
			return ErrPoolIsClosed
		}

		w, _, err := p.workerCache.Get()
		if err != nil && err != safe_queue.ErrQueueIsEmpty {
			return err
		}
		if w != nil {
			w.(*worker).work(ctx, fn)
			return nil
		}

		p.workingNumLock.Lock()
		workingNum := atomic.LoadUint32(&p.workingNum)
		if workingNum < p.capacity {
			atomic.AddUint32(&p.workingNum, 1)
			p.workingNumLock.Unlock()
			p.workerPoll.Get().(*worker).work(ctx, fn)
		}
		p.workingNumLock.Unlock()

		if p.maxWaitTaskSize == 0 {
			return ErrPollIsOverload
		}

		p.blockingNumLock.Lock()
		blockingNum := atomic.LoadUint32(&p.blockingNum)
		if blockingNum >= p.maxWaitTaskSize {
			p.blockingNumLock.Unlock()
			return ErrPollIsOverload
		}
		atomic.AddUint32(&p.blockingNum, 1)
		p.blockingNumLock.Unlock()
	}
}

func (p *Pool) Close() error {
	atomic.StoreUint32(&p.closed, 1)
	return nil
}

func (p *Pool) IsClosed() bool {
	return atomic.LoadUint32(&p.closed) != 0
}

func (p *Pool) Cap() uint32 {
	return p.capacity
}

func (p *Pool) CurrentBlockingSize() uint32 {
	return atomic.LoadUint32(&p.blockingNum)
}

func (p *Pool) CurrentWorkingNum() uint32 {
	return atomic.LoadUint32(&p.workingNum)
}
