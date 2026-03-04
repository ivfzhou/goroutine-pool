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
	"sync/atomic"
	"time"

	queue "gitee.com/ivfzhou/safe-queue"
)

const cleanWorkerInterval = time.Second

var (
	// ErrPoolIsOverload 代表池已满。
	ErrPoolIsOverload = errors.New("pool is overload")

	// ErrPoolIsClosed 代表池已关闭。
	ErrPoolIsClosed = errors.New("pool is closed")
)

// Pool 协程池结构体。
type Pool struct {
	*option
	submitNumber        int64
	closedFlag          chan struct{}
	exitBlockedTaskFlag chan struct{}
	closeOnce           sync.Once
	workers             []*worker
	blockedTasks        *queue.Queue[func()]
	modifyWorkersLock   sync.RWMutex
}

// New 新建协程池。
func New(options ...OptionFunc) *Pool {
	pool := &Pool{
		option:     loadOption(options...),
		closedFlag: make(chan struct{}),
	}
	pool.workers = make([]*worker, pool.initializedSize)
	for i := range pool.workers {
		pool.workers[i] = newWorker()
	}

	// 读取队列，并提交任务。
	if pool.maximumWaitingSize > 0 {
		pool.blockedTasks = queue.New[func()](pool.maximumWaitingSize)
		pool.exitBlockedTaskFlag = make(chan struct{})
		go func() {
			for {
				select {
				case <-pool.closedFlag:
					for {
						fn, _, err := pool.blockedTasks.Get()
						if errors.Is(err, queue.ErrQueueIsEmpty) && atomic.LoadInt64(&pool.submitNumber) == 0 {
							break
						}
						if err != nil {
							continue
						}
						for !pool.submit(fn) {
							runtime.Gosched()
						}
					}
					close(pool.exitBlockedTaskFlag)
					return
				default:
					fn, _, err := pool.blockedTasks.Get()
					if errors.Is(err, queue.ErrQueueIsEmpty) {
						time.Sleep(time.Millisecond * 10)
						continue
					}
					for !pool.submit(fn) {
						runtime.Gosched()
					}
				}
			}
		}()
	}

	// 定时清理多余工作者。
	go func() {
		ticker := time.NewTicker(cleanWorkerInterval)
		for {
			select {
			case <-ticker.C:
				if len(pool.workers)-pool.minimumIdleSize > 0 {
					go func() {
						if !pool.modifyWorkersLock.TryLock() {
							return
						}
						defer pool.modifyWorkersLock.Unlock()
						newWorkers := make([]*worker, 0, len(pool.workers))
						for _, w := range pool.workers {
							if w.isIdle(pool.maximumIdleTimeout) {
								w.close()
							} else {
								newWorkers = append(newWorkers, w)
							}
						}
						pool.workers = newWorkers
					}()
				}
			case <-pool.closedFlag:
				ticker.Stop()
				return
			}
		}
	}()

	return pool
}

// Submit 提交任务。
func (p *Pool) Submit(fn func()) error {
	atomic.AddInt64(&p.submitNumber, 1)
	defer atomic.AddInt64(&p.submitNumber, -1)

	if p.IsClosed() {
		return ErrPoolIsClosed
	}

	// 将任务提交到队列。
	if p.maximumWaitingSize > 0 && p.blockedTasks.Cap() > 0 {
		_, err := p.blockedTasks.Put(fn)
		if err == nil {
			return nil
		}
	}

	// 提交任务。
	if p.submit(fn) {
		return nil
	}

	return ErrPoolIsOverload
}

// Close 关闭协程池。
func (p *Pool) Close() {
	p.closeOnce.Do(func() {
		close(p.closedFlag)
		if p.exitBlockedTaskFlag != nil {
			<-p.exitBlockedTaskFlag
		}
		p.modifyWorkersLock.Lock()
		defer p.modifyWorkersLock.Unlock()
		for _, w := range p.workers {
			w.close()
		}
		p.workers = nil
		p.blockedTasks = nil
		p.option = nil
	})
}

// IsClosed 协程池是否已关闭。
func (p *Pool) IsClosed() bool {
	select {
	case <-p.closedFlag:
		return true
	default:
		return false
	}
}

// Cap 最大工作者数。
func (p *Pool) Cap() int {
	return p.maximumSize
}

// WaitingTaskSize 当前阻塞的任务数。
func (p *Pool) WaitingTaskSize() int {
	if p.blockedTasks == nil {
		return 0
	}
	return int(p.blockedTasks.Len())
}

// WorkerSize 当前运行的协程数。
func (p *Pool) WorkerSize() int {
	p.modifyWorkersLock.RLock()
	defer p.modifyWorkersLock.RUnlock()
	return len(p.workers)
}

// 提交任务。
func (p *Pool) submit(fn func()) bool {
	p.modifyWorkersLock.RLock()

	// 有工作者，就发给工作者处理。
	if len(p.workers) > 0 {
		for _, w := range p.workers {
			select {
			case w.channel <- fn:
				p.modifyWorkersLock.RUnlock()
				return true
			default:
			}
		}
	}

	// 没有工作者，或者没有空闲的工作者，且工作者数量达到上限，就提交失败。
	if len(p.workers) >= p.maximumSize {
		p.modifyWorkersLock.RUnlock()
		return false
	}
	p.modifyWorkersLock.RUnlock()

	p.modifyWorkersLock.Lock()
	defer p.modifyWorkersLock.Unlock()

	// 工作者数量没有达到上限，创建工作者执行任务。
	if len(p.workers) < p.maximumSize {
		w := newWorker()
		p.workers = append(p.workers, w)
		w.start()
		w.channel <- fn
		return true
	}

	return false
}
