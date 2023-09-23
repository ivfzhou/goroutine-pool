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
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pool "gitee.com/ivfzhou/goroutine-pool"
)

func TestCommon(t *testing.T) {
	p := pool.New(pool.WithMaxSize(2))
	if p.Cap() != 2 {
		t.Fatal("cap != 2")
	}
	if p.WorkerSize() != 0 {
		t.Fatal("worker != 0")
	}
	if p.WaitingTaskSize() != 0 {
		t.Fatal("block != 0")
	}
	if p.IsClosed() {
		t.Fatal("p is isClosed")
	}
	ch := make(chan struct{})
	fn := func() {
		<-ch
	}
	err := p.Submit(fn)
	if err != nil {
		t.Fatal(err)
	}
	err = p.Submit(fn)
	if err != nil {
		t.Fatal(err)
	}
	close(ch)
	p.Submit(func() {
		panic("nil")
	})
	time.Sleep(3 * time.Second)
	p.Close()
	if !p.IsClosed() {
		t.Fatal("p is not isClosed")
	}
}

func TestPanic(t *testing.T) {
	p := pool.New(pool.WithMaxSize(1))
	err := p.Submit(func() {
		fmt.Println("run")
		panic("1")
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	err = p.Submit(func() {
		fmt.Println("run again")
		panic("1")
	})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	p.Close()
}

func BenchmarkPool(b *testing.B) {
	p := pool.New(pool.WithInitSize(1024), pool.WithMinIdleSize(1024), pool.WithMaxWaitingSize(1024))
	wg := sync.WaitGroup{}
	count := uint64(0)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		wg.Add(1)
		_ = p.Submit(func() {
			atomic.AddUint64(&count, 1)
			wg.Done()
		})
	}
	wg.Wait()
	b.StopTimer()
	b.Log(count)
}
