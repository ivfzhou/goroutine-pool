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
	"sync/atomic"
	"testing"
	"time"

	pool "gitee.com/ivfzhou/goroutine-pool"
)

func TestPool(t *testing.T) {
	{
		size := 5
		minimumSize := 1
		p := pool.New(pool.WithInitSize(minimumSize), pool.WithMaxSize(size), pool.WithMaxIdleTimeout(time.Minute),
			pool.WithMinIdleSize(size))
		if p.Cap() != size {
			t.Fatal("pool cap is not", size)
		}
		waitChannel := make(chan struct{})
		fn := func() {
			<-waitChannel
		}
		for range size {
			_ = p.Submit(fn)
		}
		if p.WorkerSize() != size {
			t.Fatal("waiting task size is not", size)
		}
		close(waitChannel)
		p.Close()
	}

	{
		size := 5
		minimumSize := 1
		p := pool.New(pool.WithInitSize(minimumSize), pool.WithMaxSize(size), pool.WithMaxIdleTimeout(3*time.Second))
		waitChannel := make(chan struct{})
		fn := func() {
			<-waitChannel
		}
		for range size {
			_ = p.Submit(fn)
		}
		close(waitChannel)
		time.Sleep(10 * time.Second)
		if p.WorkerSize() != minimumSize {
			t.Fatal("waiting task size is not", minimumSize, p.WorkerSize())
		}
		p.Close()
	}

	{
		size := 5
		minimumSize := 1
		p := pool.New(pool.WithInitSize(minimumSize), pool.WithMaxSize(size), pool.WithMaxIdleTimeout(time.Minute),
			pool.WithMinIdleSize(size))
		waitChannel := make(chan struct{})
		fn := func() {
			<-waitChannel
		}
		for range size * 2 {
			_ = p.Submit(fn)
		}
		err := p.Submit(fn)
		if !errors.Is(err, pool.ErrPoolIsOverload) {
			t.Fatal("waiting task size is not overload")
		}
		close(waitChannel)
		p.Close()
	}

	{
		size := 5
		minimumSize := 1
		p := pool.New(pool.WithInitSize(minimumSize), pool.WithMaxSize(size), pool.WithMaxIdleTimeout(time.Minute),
			pool.WithMinIdleSize(size), pool.WithMaxWaitingSize(size))
		waitChannel := make(chan struct{})
		result := int64(0)
		fn := func() {
			<-waitChannel
			atomic.AddInt64(&result, 1)
		}
		expectedResult := int64(0)
		for range size * 3 {
			err := p.Submit(fn)
			if err == nil {
				expectedResult++
			}
		}
		if p.WaitingTaskSize() != size {
			t.Fatal("waiting task size is not", size, p.WaitingTaskSize())
		}
		close(waitChannel)
		p.Close()
		time.Sleep(10 * time.Second)
		if expectedResult-1 != atomic.LoadInt64(&result) {
			t.Fatal("result is unexpected", result, expectedResult)
		}
	}
}
