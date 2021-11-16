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

func WithPanicHandler(fn func(p interface{})) optionFunc {
	return func(o *option) {
		o.panicHandler = fn
	}
}

func WithCapacity(capacity uint32) optionFunc {
	return func(o *option) {
		o.capacity = capacity
	}
}

func WithIdleTimeout(t time.Duration) optionFunc {
	return func(o *option) {
		o.idleTimeout = t
	}
}

func WithMaxWaitTaskSize(n uint32) optionFunc {
	return func(o *option) {
		o.maxWaitTaskSize = n
	}
}
