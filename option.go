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
	defaultMaximumSize = math.MaxUint32
)

type option struct {
	maximumSize        int
	minimumIdleSize    int
	initializedSize    int
	maximumIdleTimeout time.Duration
	maximumWaitingSize uint32
}

type OptionFunc func(*option)

func loadOption(options ...OptionFunc) *option {
	o := &option{}
	for _, fn := range options {
		fn(o)
	}

	if o.maximumSize <= 0 {
		o.maximumSize = defaultMaximumSize
	}
	if o.maximumIdleTimeout <= 0 {
		o.maximumIdleTimeout = defaultIdleTimeout
	}
	if o.initializedSize > o.maximumSize {
		o.initializedSize = o.maximumSize
	}
	if o.minimumIdleSize > o.maximumSize {
		o.minimumIdleSize = o.maximumSize
	}
	if o.minimumIdleSize <= 0 {
		o.minimumIdleSize = o.initializedSize
	}

	return o
}

// WithMaxSize 协程池中最大工作者数量。
func WithMaxSize(size int) OptionFunc {
	return func(o *option) {
		if size > 0 {
			o.maximumSize = size
		}
	}
}

// WithMinIdleSize 协程池最少工作者数量。
func WithMinIdleSize(size int) OptionFunc {
	return func(o *option) {
		if size > 0 {
			o.minimumIdleSize = size
		}
	}
}

// WithInitSize 协程池初始工作者数量。
func WithInitSize(size int) OptionFunc {
	return func(o *option) {
		if size > 0 {
			o.initializedSize = size
		}
	}
}

// WithMaxIdleTimeout 工作者最大空闲回收时间。
func WithMaxIdleTimeout(d time.Duration) OptionFunc {
	return func(o *option) {
		if d > 0 {
			o.maximumIdleTimeout = d
		}
	}
}

// WithMaxWaitingSize 任务最大等待个数。
func WithMaxWaitingSize(size int) OptionFunc {
	return func(o *option) {
		if size > 0 {
			if size > math.MaxUint32 {
				size = math.MaxUint32
			}
			o.maximumWaitingSize = uint32(size)
		}
	}
}
