package goroutine_pool_test

import (
	"context"
	"testing"
	"time"

	goroutine_pool "gitee.com/ivfzhou/goroutine-pool"
)

func taskFunc(ctx context.Context) {
	time.Sleep(time.Millisecond * 200)
	println("done")
}

func TestCommon(t *testing.T) {
	pool := goroutine_pool.New(goroutine_pool.WithCapacity(3))
	for i := 0; i < 3; i++ {
		err := pool.Submit(context.Background(), taskFunc)
		if err != nil {
			t.Fatal(err)
		}
	}
	err := pool.Submit(context.Background(), taskFunc)
	if err != goroutine_pool.ErrPollIsOverload {
		t.Fatal("err != Overload")
	}
	if pool.CurrentWorkingNum() != 3 {
		t.Fatal("workingNum != 3")
	}
	if pool.Cap() != 3 {
		t.Fatal("cap != 3")
	}
	if pool.CurrentBlockingSize() != 0 {
		t.Fatal("blockingSize != 0")
	}
	if pool.IsClosed() {
		t.Fatal("closed == true")
	}


	pool = goroutine_pool.New(goroutine_pool.WithCapacity(3), goroutine_pool.WithIdleTimeout(time.Millisecond*500))

	time.Sleep(time.Hour)
}
