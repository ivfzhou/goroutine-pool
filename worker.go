package goroutine_pool

import (
	"context"
	"sync/atomic"
	"time"
)

type task struct {
	ctx context.Context
	fn  func(context.Context)
}

type worker struct {
	tasks            chan *task
	latestAccessTime time.Time
}

func (w *worker) work(ctx context.Context, fn func(context.Context)) {
	w.tasks <- &task{ctx, fn}
}

func (w *worker) start(pool *Pool) {
	w.tasks = make(chan *task, taskCache)
	go func() {
		defer func() {
			_ = atomic.AddUint32(&pool.workingNum, negativeOne)
			if pool.panicHandler != nil {
				if p := recover(); p != nil {
					pool.panicHandler(p)
				}
			}
		}()

		for v := range w.tasks {
			v.fn(v.ctx)
			w.latestAccessTime = time.Now()
			pool.cache <- w
		}
	}()
}

func (w *worker) close() {
	close(w.tasks)
}

func (w *worker) isTimeout(idleTimeout time.Duration) bool {
	if time.Now().After(w.latestAccessTime.Add(idleTimeout)) {
		return true
	}
	return false
}
