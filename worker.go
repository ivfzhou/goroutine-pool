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
	tasks      chan *task
	done       *time.Timer
	nextWorker *worker
}

func (w *worker) work(ctx context.Context, fn func(ctx context.Context)) {
	w.tasks <- &task{ctx, fn}
}

func (w *worker) start() {
	w.tasks = make(chan *task, taskCache)
	w.done = time.NewTimer(w.idleTimeout)
	go func() {
		defer func() {
			defer func() {
				_ = atomic.AddUint32(&w.workingNum, ^uint32(1)+1)
				_ = w.done.Stop()
			}()
			if w.panicHandler != nil {
				if p := recover(); p != nil {
					w.panicHandler(p)
				}
			}
		}()

	L1:
		for {
			select {
			case v := <-w.tasks:
				v.fn(v.ctx)
				_ = w.done.Stop()
				_ = w.done.Reset(w.idleTimeout)
				w.workerCache.PushTail(w)
			case <-w.done.C:
				goto L2
			case <-w.closed:
				goto L2
			}
		}
	L2:
		select {
		case v := <-w.tasks:
			v.fn(v.ctx)
			_ = w.done.Stop()
			_ = w.done.Reset(w.idleTimeout)
			goto L1
		default:
		}
	}()
}

func (w *worker) isClosed() bool {
	select {
	case <-w.done.C:
		return true
	default:
		return false
	}
}
