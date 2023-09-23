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
	"math"
	"time"
)

const (
	defaultIdleTimeout = 5 * time.Second
	defaultMaxSize     = math.MaxUint32
)

type option struct {
	maxSize        int
	minIdleSize    int
	initSize       int
	maxIdleTimeout time.Duration
	maxWaitingSize int
}

type optionFunc func(*option)

func loadOption(opts ...optionFunc) *option {
	o := &option{}
	for _, fn := range opts {
		fn(o)
	}

	if o.maxSize <= 0 {
		o.maxSize = defaultMaxSize
	}
	if o.maxIdleTimeout <= 0 {
		o.maxIdleTimeout = defaultIdleTimeout
	}
	if o.initSize > o.maxSize {
		o.initSize = o.maxSize
	}
	if o.minIdleSize > o.maxSize {
		o.minIdleSize = o.maxSize
	}

	return o
}

// WithMaxSize 协程池最大数量。
func WithMaxSize(maxSize int) optionFunc {
	return func(o *option) {
		if maxSize > 0 {
			o.maxSize = maxSize
		}
	}
}

// WithMinIdleSize 协程池最少数量。
func WithMinIdleSize(minIdleSize int) optionFunc {
	return func(o *option) {
		if minIdleSize > 0 {
			o.minIdleSize = minIdleSize
		}
	}
}

// WithInitSize 协程池初始数量。
func WithInitSize(initSize int) optionFunc {
	return func(o *option) {
		if initSize > 0 {
			o.initSize = initSize
		}
	}
}

// WithMaxIdleTimeout 协程最大空间回收时间。
func WithMaxIdleTimeout(maxIdleTimeout time.Duration) optionFunc {
	return func(o *option) {
		if maxIdleTimeout > 0 {
			o.maxIdleTimeout = maxIdleTimeout
		}
	}
}

// WithMaxWaitingSize 任务最大等待个数。
func WithMaxWaitingSize(maxWaitingSize int) optionFunc {
	return func(o *option) {
		if maxWaitingSize > 0 {
			o.maxWaitingSize = maxWaitingSize
		}
	}
}
