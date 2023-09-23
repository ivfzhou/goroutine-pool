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
	"runtime"
	"sync/atomic"
	"time"
)

var taskCache = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

type worker struct {
	startIdleTime time.Time
	running       uint32
	c             chan func()
}

func newWorker() *worker {
	return &worker{startIdleTime: time.Now().Add(defaultIdleTimeout), c: make(chan func(), taskCache)}
}

func (w *worker) start() {
	go func() {
		defer func() {
			if recover() != nil {
				w.start()
			}
		}()
		for fn := range w.c {
			atomic.StoreUint32(&w.running, 1)
			fn()
			w.startIdleTime = time.Now()
			atomic.StoreUint32(&w.running, 0)
		}
	}()
}

func (w *worker) isIdle(idleTimeout time.Duration) bool {
	if atomic.LoadUint32(&w.running) <= 0 && time.Now().After(w.startIdleTime.Add(idleTimeout)) {
		return true
	}
	return false
}
