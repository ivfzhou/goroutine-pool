package goroutine_pool_test

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitee.com/ivfzhou/goroutine-pool"

	"github.com/panjf2000/ants/v2"
)

func TestPool(t *testing.T) {
	pool := goroutine_pool.New(goroutine_pool.WithCapacity(20000), goroutine_pool.WithMaxWaitTaskSize(1000))
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
	_ = pool.Close()
	if atomic.LoadUint64(&count) != 300000 {
		t.Fatal("count != 300000", count)
	}
}

const times = 1000000

// 2716236634
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

//507979034
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

//690966338
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
