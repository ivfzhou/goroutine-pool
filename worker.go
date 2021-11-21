package goroutine_pool

import (
	"context"
	"sync/atomic"
	"time"
)

var closedChan = make(chan struct{})

func init() {
	close(closedChan)
}

type task struct {
	ctx context.Context
	fn  func(context.Context)
}

type worker struct {
	tasks  chan *task
	timer  *time.Timer
	closed chan struct{}
}

func (w *worker) work(ctx context.Context, fn func(ctx context.Context)) {
	if w.isClosed() {
		return
	}
	w.tasks <- &task{ctx, fn}
}

func (w *worker) start(pool *Pool) {
	w.tasks = make(chan *task, taskCache)
	w.timer = time.NewTimer(pool.idleTimeout)
	go func() {
		defer func() {
			defer func() {
				_ = atomic.AddUint32(&pool.workingNum, negativeOne)
				_ = w.timer.Stop()
				w.timer = nil
				w.tasks = nil
			}()
			if pool.panicHandler != nil {
				if p := recover(); p != nil {
					pool.panicHandler(p)
				}
			}
		}()

		for {
			select {
			case v := <-w.tasks:
				v.fn(v.ctx)
				w.resetTimer(pool.idleTimeout)
				pool.cond.L.Lock()
				pool.pushTail(w)
				pool.cond.L.Unlock()
				pool.cond.Signal()
			case <-w.timer.C:
				w.closed = closedChan
				select {
				case v := <-w.tasks:
					v.fn(v.ctx)
					w.resetTimer(pool.idleTimeout)
				default:
				}
				return
			case <-pool.closed:
				select {
				case v := <-w.tasks:
					v.fn(v.ctx)
				default:
				}
				return
			}
		}
	}()
}

func (w *worker) isClosed() bool {
	select {
	case <-w.closed:
		return true
	default:
		return false
	}
}

func (w *worker) runTask(idleTimeout time.Duration) bool {
	select {
	case v := <-w.tasks:
		v.fn(v.ctx)
		w.resetTimer(idleTimeout)
		return true
	default:
		return false
	}
}

func (w *worker) resetTimer(t time.Duration) {
	_ = w.timer.Stop()
	_ = w.timer.Reset(t)
}
