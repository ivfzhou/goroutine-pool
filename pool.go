/*
 * Copyright (c) 2023 ivfzhou
 * goroutine-pool is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package goroutine_pool

import (
	"errors"
	"runtime"
	"sync"
	"time"

	queue "gitee.com/ivfzhou/safe-queue"
)

// Pool 协程池结构体。
type Pool struct {
	*option
	isClosed     chan struct{}
	cacheWorkers []*worker
	blockTasks   *queue.Queue[func()]
	lock         sync.RWMutex
}

// New 新建协程池。opts 协程池参数。
func New(opts ...OptionFunc) *Pool {
	p := &Pool{
		option:   loadOption(opts...),
		isClosed: make(chan struct{}),
	}
	p.cacheWorkers = make([]*worker, p.initSize)
	for i := range p.cacheWorkers {
		p.cacheWorkers[i] = newWorker()
	}
	p.blockTasks = queue.New[func()](uint32(p.maxWaitingSize))
	p.start()
	return p
}

// Submit 提交任务。
func (p *Pool) Submit(fn func()) error {
	if p.IsClosed() {
		return ErrPoolIsClosed
	}

	if p.maxWaitingSize > 0 && p.blockTasks.Len() > 0 {
		_, err := p.blockTasks.Put(fn)
		if errors.Is(err, queue.ErrQueueIsFull) {
			return ErrPoolIsOverload
		}
		return nil
	}

	if p.submit(fn) {
		return nil
	}

	return ErrPoolIsOverload
}

// Close 关闭协程池。
func (p *Pool) Close() {
	close(p.isClosed)
}

// IsClosed 协程池是否已关闭。
func (p *Pool) IsClosed() bool {
	select {
	case <-p.isClosed:
		return true
	default:
		return false
	}
}

// Cap 最大协程数。
func (p *Pool) Cap() int {
	return p.maxSize
}

// WaitingTaskSize 当前阻塞的任务数。
func (p *Pool) WaitingTaskSize() int {
	return int(p.blockTasks.Len())
}

// WorkerSize 当前运行的协程数。
func (p *Pool) WorkerSize() int {
	return len(p.cacheWorkers)
}

func (p *Pool) start() {
	if p.maxWaitingSize > 0 {
		go func() {
			fn, _ := p.blockTasks.MustGet()
			for !p.submit(fn) {
				runtime.Gosched()
			}
		}()
	}
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		for range ticker.C {
			num := len(p.cacheWorkers) - p.minIdleSize
			if num > 0 {
				p.lock.Lock()
				count := 0
				newCache := make([]*worker, 0, len(p.cacheWorkers))
				for _, w := range p.cacheWorkers {
					if count >= num {
						break
					}
					if w.isIdle(p.maxIdleTimeout) {
						count++
						close(w.c)
					} else {
						newCache = append(newCache, w)
					}
				}
				p.cacheWorkers = newCache
				p.lock.Unlock()
			}
		}
	}()
}

func (p *Pool) submit(fn func()) bool {
	p.lock.RLock()
	if len(p.cacheWorkers) > 0 {
		for _, w := range p.cacheWorkers {
			select {
			case w.c <- fn:
				p.lock.RUnlock()
				return true
			default:
			}
		}
	}
	if len(p.cacheWorkers) >= p.maxSize {
		return false
	}
	p.lock.RUnlock()

	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.cacheWorkers) < p.maxSize {
		w := newWorker()
		p.cacheWorkers = append(p.cacheWorkers, w)
		w.start()
		w.c <- fn
		return true
	}

	return false
}
