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

package goroutine_pool_test

import (
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pool "gitee.com/ivfzhou/goroutine-pool"
)

// waitFor 在 timeout 时间内轮询等待条件成立，避免使用固定 sleep 导致的偶发失败。
func waitFor(t *testing.T, timeout time.Duration, desc string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("等待超时：%s", desc)
}

// TestCap 验证 Cap 返回最大工作者数。
func TestCap(t *testing.T) {
	p := pool.New(pool.WithMaxSize(10))
	defer p.Close()

	if got := p.Cap(); got != 10 {
		t.Fatalf("Cap() = %d, want 10", got)
	}
}

// TestDefaultOptions 验证默认选项下池的基本行为。
func TestDefaultOptions(t *testing.T) {
	p := pool.New()
	defer p.Close()

	if got := p.Cap(); got != int(math.MaxUint32) {
		t.Fatalf("默认 Cap() = %d, want %d", got, int(math.MaxUint32))
	}
	if got := p.WorkerSize(); got != 0 {
		t.Fatalf("默认 WorkerSize() = %d, want 0", got)
	}
	if got := p.WaitingTaskSize(); got != 0 {
		t.Fatalf("默认 WaitingTaskSize() = %d, want 0", got)
	}
	if p.IsClosed() {
		t.Fatal("新建的池不应已关闭")
	}

	var n int64
	if err := p.Submit(func() { atomic.AddInt64(&n, 1) }); err != nil {
		t.Fatalf("Submit() err = %v", err)
	}
	waitFor(t, time.Second, "任务执行", func() bool { return atomic.LoadInt64(&n) == 1 })
}

// TestInitializedWorkersExecute 验证初始化的 worker 会真正执行任务。
func TestInitializedWorkersExecute(t *testing.T) {
	const n = 3
	p := pool.New(pool.WithInitSize(n), pool.WithMaxSize(n))
	defer p.Close()

	var count int64
	for i := 0; i < n; i++ {
		if err := p.Submit(func() { atomic.AddInt64(&count, 1) }); err != nil {
			t.Fatalf("第 %d 次 Submit() err = %v", i, err)
		}
	}
	waitFor(t, time.Second, "初始化 worker 执行任务", func() bool {
		return atomic.LoadInt64(&count) == n
	})
}

// TestWorkerScaleUp 验证在持续阻塞负载下 worker 能扩容到最大值。
func TestWorkerScaleUp(t *testing.T) {
	const max = 5
	p := pool.New(pool.WithInitSize(1), pool.WithMaxSize(max), pool.WithMaxIdleTimeout(time.Minute))
	defer p.Close()

	release := make(chan struct{})
	fn := func() { <-release }

	// 持续提交阻塞任务，直到池满载返回 overload；此时 worker 数必然达到 max。
	var overloaded bool
	for i := 0; i < max*4; i++ {
		err := p.Submit(fn)
		if errors.Is(err, pool.ErrPoolIsOverload) {
			overloaded = true
			break
		}
		if err != nil {
			t.Fatalf("Submit() err = %v", err)
		}
	}
	if !overloaded {
		t.Fatal("期望满载后返回 ErrPoolIsOverload")
	}
	if got := p.WorkerSize(); got != max {
		t.Fatalf("WorkerSize() = %d, want %d", got, max)
	}
	close(release)
}

// TestWorkerScaleDown 验证空闲 worker 会被回收至最少空闲数，且不会继续下降。
func TestWorkerScaleDown(t *testing.T) {
	const max = 5
	const minIdle = 2
	p := pool.New(pool.WithInitSize(1), pool.WithMaxSize(max),
		pool.WithMinIdleSize(minIdle), pool.WithMaxIdleTimeout(200*time.Millisecond))
	defer p.Close()

	release := make(chan struct{})
	fn := func() { <-release }
	// 先扩容到 max：持续提交阻塞任务直到池满载返回 overload，此时 worker 数必然达到 max。
	var overloaded bool
	for i := 0; i < max*4; i++ {
		err := p.Submit(fn)
		if errors.Is(err, pool.ErrPoolIsOverload) {
			overloaded = true
			break
		}
		if err != nil {
			t.Fatalf("Submit() err = %v", err)
		}
	}
	if !overloaded {
		t.Fatal("期望满载后返回 ErrPoolIsOverload")
	}
	if got := p.WorkerSize(); got != max {
		t.Fatalf("扩容后 WorkerSize() = %d, want %d", got, max)
	}
	close(release)

	// 等待回收至 minIdle。
	waitFor(t, 5*time.Second, "worker 收缩到 minIdle", func() bool { return p.WorkerSize() == minIdle })
	// 再观察一段时间，确认不会继续收缩到 minIdle 以下。
	time.Sleep(time.Second)
	if got := p.WorkerSize(); got != minIdle {
		t.Fatalf("收缩后 WorkerSize() = %d, want %d", got, minIdle)
	}
}

// TestWaitingQueueBounded 验证等待队列长度不会超过 maximumWaitingSize。
// 该用例用于防止底层无锁队列将容量向上取整为 2 的幂而突破上限的问题。
func TestWaitingQueueBounded(t *testing.T) {
	const max = 5
	const waiting = 5
	p := pool.New(pool.WithMaxSize(max), pool.WithMaxIdleTimeout(time.Minute),
		pool.WithMaxWaitingSize(waiting))
	defer p.Close()

	release := make(chan struct{})
	fn := func() { <-release }

	for i := 0; i < max*4; i++ {
		_ = p.Submit(fn)
	}
	if got := p.WaitingTaskSize(); got > waiting {
		t.Fatalf("WaitingTaskSize() = %d, 超过上限 %d", got, waiting)
	}
	close(release)
}

// TestSubmitAllTasksExecute 验证通过等待队列缓冲的任务最终全部执行且只执行一次。
func TestSubmitAllTasksExecute(t *testing.T) {
	const max = 10
	const taskNum = 100
	p := pool.New(pool.WithMaxSize(max), pool.WithMaxWaitingSize(taskNum))
	defer p.Close()

	var count int64
	for i := 0; i < taskNum; i++ {
		if err := p.Submit(func() { atomic.AddInt64(&count, 1) }); err != nil {
			t.Fatalf("第 %d 次 Submit() err = %v", i, err)
		}
	}
	waitFor(t, 5*time.Second, "所有任务执行完成", func() bool {
		return atomic.LoadInt64(&count) == taskNum
	})
}

// TestSubmitAfterClose 验证关闭后提交任务返回 ErrPoolIsClosed。
func TestSubmitAfterClose(t *testing.T) {
	p := pool.New(pool.WithMaxSize(2))
	p.Close()

	if err := p.Submit(func() {}); !errors.Is(err, pool.ErrPoolIsClosed) {
		t.Fatalf("Submit() err = %v, want ErrPoolIsClosed", err)
	}
}

// TestCloseIdempotent 验证 Close 可重复调用且不 panic。
func TestCloseIdempotent(t *testing.T) {
	p := pool.New(pool.WithMaxSize(2))
	p.Close()
	p.Close()

	if !p.IsClosed() {
		t.Fatal("IsClosed() = false, want true")
	}
}

// TestStateAfterClose 验证 Close 后查询方法仍可安全调用（不 panic）。
func TestStateAfterClose(t *testing.T) {
	p := pool.New(pool.WithMaxSize(10), pool.WithMaxWaitingSize(5))
	p.Close()

	if got := p.Cap(); got != 10 {
		t.Fatalf("Close 后 Cap() = %d, want 10", got)
	}
	if got := p.WorkerSize(); got != 0 {
		t.Fatalf("Close 后 WorkerSize() = %d, want 0", got)
	}
	if got := p.WaitingTaskSize(); got != 0 {
		t.Fatalf("Close 后 WaitingTaskSize() = %d, want 0", got)
	}
}

// TestCloseDrainsQueuedTasks 验证 Close 会排空等待队列，所有已提交任务最终都会执行。
func TestCloseDrainsQueuedTasks(t *testing.T) {
	const max = 2
	const total = max + 5
	p := pool.New(pool.WithMaxSize(max), pool.WithMaxWaitingSize(total))

	var done int64
	for i := 0; i < total; i++ {
		if err := p.Submit(func() { atomic.AddInt64(&done, 1) }); err != nil {
			t.Fatalf("第 %d 次 Submit() err = %v", i, err)
		}
	}
	p.Close()

	waitFor(t, 5*time.Second, "Close 后排空队列中的任务", func() bool {
		return atomic.LoadInt64(&done) == total
	})
}

// TestPanicRecovery 验证任务 panic 不会导致协程池崩溃，后续任务仍能执行。
func TestPanicRecovery(t *testing.T) {
	p := pool.New(pool.WithMaxSize(2))
	defer p.Close()

	if err := p.Submit(func() { panic("boom") }); err != nil {
		t.Fatalf("Submit() err = %v", err)
	}

	done := make(chan struct{})
	if err := p.Submit(func() { close(done) }); err != nil {
		t.Fatalf("Submit() err = %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("panic 后任务未能正常执行")
	}
}

// TestConcurrentSubmit 验证并发提交时无数据竞争、任务不丢失且不重复执行。
func TestConcurrentSubmit(t *testing.T) {
	const max = 20
	const taskNum = 200
	p := pool.New(pool.WithMaxSize(max), pool.WithMaxWaitingSize(taskNum))
	defer p.Close()

	var executed, accepted int64
	var wg sync.WaitGroup
	for i := 0; i < taskNum; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := p.Submit(func() { atomic.AddInt64(&executed, 1) })
			if err == nil {
				atomic.AddInt64(&accepted, 1)
			} else if !errors.Is(err, pool.ErrPoolIsOverload) {
				t.Errorf("Submit() err = %v", err)
			}
		}()
	}
	wg.Wait()

	// 被接受的任务必须全部执行完毕，且执行次数与被接受数一致。
	waitFor(t, 10*time.Second, "被接受的任务全部执行", func() bool {
		return atomic.LoadInt64(&executed) == atomic.LoadInt64(&accepted)
	})
}
