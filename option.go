package goroutine_pool

import "time"

type option struct {
	capacity        uint32
	idleTimeout     time.Duration
	maxWaitTaskSize uint32
	panicHandler    func(p interface{})
}

type optionFunc func(*option)

func loadOption(opts ...optionFunc) *option {
	o := new(option)
	for _, fn := range opts {
		fn(o)
	}
	return o
}

// WithPanicHandler 设置协程池协程执行任务触发恐慌时的代理函数。如果未设置，执行任务时恐慌将往外抛出。
//
// fn 恐慌代理函数。
//   p 恐慌传递的值。
func WithPanicHandler(fn func(p interface{})) optionFunc {
	return func(o *option) {
		o.panicHandler = fn
	}
}

// WithCapacity 设置协程池最大协程数。若不设置，默认为 2^32-1。
//
// capacity 协程池最大协程数
func WithCapacity(capacity uint32) optionFunc {
	return func(o *option) {
		o.capacity = capacity
	}
}

// WithIdleTimeout 设置协程最大空闲时间，超过时间的协程将自动销毁，默认为 2s。
//
// t 协程最大空闲时间。
func WithIdleTimeout(t time.Duration) optionFunc {
	return func(o *option) {
		o.idleTimeout = t
	}
}

// WithMaxWaitTaskSize 设置阻塞待执行任务最大数，默认为零表示不阻塞。
//
// n 阻塞待执行任务最大数。
func WithMaxWaitTaskSize(n uint32) optionFunc {
	return func(o *option) {
		o.maxWaitTaskSize = n
	}
}
