package goroutine_pool_test

import (
	"context"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitee.com/ivfzhou/goroutine-pool"

	"github.com/panjf2000/ants/v2"
)

func TestCommon(t *testing.T) {
	pool := goroutine_pool.New(goroutine_pool.WithCapacity(1))
	if pool.Cap() != 1 {
		t.Fatal("cap != 1")
	}
	if pool.CurrentWorkingNum() != 0 {
		t.Fatal("worker != 0")
	}
	if pool.CurrentBlockingSize() != 0 {
		t.Fatal("block != 0")
	}
	if pool.IsClosed() {
		t.Fatal("pool is closed")
	}
	ctx := context.Background()
	ch := make(chan struct{})
	fn := func(c context.Context) {
		if ctx != c {
			t.Error("ctx != c")
		}
		<-ch
	}
	err := pool.Submit(ctx, fn)
	if err != nil {
		t.Fatal(err)
	}
	err = pool.Submit(ctx, fn)
	if err != goroutine_pool.ErrPoolIsOverload {
		t.Fatal("err != ErrPoolIsOverload")
	}
	close(ch)
	pool.Close()
	if !pool.IsClosed() {
		t.Fatal("pool is not closed")
	}
	time.Sleep(time.Second)

	pool = goroutine_pool.New(goroutine_pool.WithMaxWaitTaskSize(1), goroutine_pool.WithCapacity(1))
	ch = make(chan struct{})
	err = pool.Submit(ctx, func(ctx context.Context) {
		<-ch
	})
	if err != nil {
		t.Fatal(err)
	}
	if pool.CurrentWorkingNum() != 1 {
		t.Fatal("work != 1")
	}
	go func() {
		err = pool.Submit(ctx, func(ctx context.Context) {})
		if err != nil {
			t.Error(err)
		}
	}()
	runtime.Gosched()
	if pool.CurrentBlockingSize() != 1 {
		t.Error("block != 1")
	}
	close(ch)
}

func TestPool(t *testing.T) {
	pool := goroutine_pool.New()
	wg := sync.WaitGroup{}
	ctx := context.Background()
	count := uint64(0)
	for i := 0; i < 300000; i++ {
		wg.Add(1)
		_ = pool.Submit(ctx, func(ctx context.Context) {
			atomic.AddUint64(&count, 1)
			time.Sleep(time.Millisecond * 50)
			wg.Done()
		})
	}
	wg.Wait()
	pool.Close()
	if atomic.LoadUint64(&count) != 300000 {
		t.Fatal("count != 300000", count)
	}
}

const times = 10000000

//7532518536
func BenchmarkPool(b *testing.B) {
	pool := goroutine_pool.New()
	wg := sync.WaitGroup{}
	ctx := context.Background()
	count := uint64(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(times)
		for j := 0; j < times; j++ {
			_ = pool.Submit(ctx, func(ctx context.Context) {
				defer wg.Done()
				time.Sleep(time.Millisecond * 50)
				atomic.AddUint64(&count, 1)
			})
		}
	}
	wg.Wait()
	b.StopTimer()
}

//5175545706
func BenchmarkNormal(b *testing.B) {
	wg := sync.WaitGroup{}
	count := uint64(0)
	ctx := context.Background()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(times)
		for j := 0; j < times; j++ {
			go func(ctx context.Context) {
				defer wg.Done()
				time.Sleep(time.Millisecond * 50)
				atomic.AddUint64(&count, 1)
			}(ctx)
		}
	}
	wg.Wait()
	b.StopTimer()
}

//6382175158
func BenchmarkAnts(b *testing.B) {
	pool, _ := ants.NewPool(math.MaxUint32)
	wg := sync.WaitGroup{}
	count := uint64(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(times)
		for j := 0; j < times; j++ {
			_ = pool.Submit(func() {
				defer wg.Done()
				time.Sleep(time.Millisecond * 50)
				atomic.AddUint64(&count, 1)
			})
		}
	}
	wg.Wait()
	b.StopTimer()
}
