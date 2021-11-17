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
	*Pool
	tasks chan *task
	done  *time.Ticker
}

func (w *worker) work(ctx context.Context, fn func(ctx context.Context)) {
	w.tasks <- &task{ctx, fn}
}

func (w *worker) start() {
	w.tasks = make(chan *task, taskCache)
	w.done = time.NewTicker(w.idleTimeout)
	go func() {
		defer func() {
			defer func() {
				atomic.AddUint32(&w.workingNum, ^uint32(1)+1)
				w.Pool.workerPoll.Put(w)
				w.reset()
			}()
			if w.panicHandler != nil {
				if p := recover(); p != nil {
					w.panicHandler(p)
				}
			}
		}()

		for {
			select {
			case v := <-w.tasks:
				v.fn(v.ctx)
				_, _ = w.Pool.workerCache.Put(w)
				w.done.Reset(w.idleTimeout)
			case <-w.done.C:
				goto L1
			case <-w.closed:
				goto L1
			}
		}
	L1:
		if v, ok := <-w.tasks; ok {
			v.fn(v.ctx)
		}
	}()
}

func (w *worker) reset() {
	w.Pool = nil
	w.tasks = nil
	w.done = nil
}
