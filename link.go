package goroutine_pool

import "sync"

type node struct {
	value          interface{}
	previous, next *node
}

type Link struct {
	lock       sync.Mutex
	head, tail *node
}

func (l *Link) GetHead(fn func(v interface{}) bool) interface{} {
	l.lock.Lock()
	defer l.lock.Unlock()
	for {
		if l.head == nil {
			return nil
		}
		v := l.head.value
		l.head = l.head.next
		if !fn(v) {
			continue
		}
		return v
	}
}

func (l *Link) CheckHead(fn func(v interface{}) bool) {
	l.lock.Lock()
	defer l.lock.Unlock()
	for {
		if l.head == nil {
			return
		}
		v := l.head.value
		if fn(v) {
			return
		} else {
			l.head = l.head.next
		}
	}
}

func (l *Link) PushTail(v interface{}) {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.head == nil {
		l.head = &node{v, nil, nil}
	} else if l.tail == nil {
		l.tail = &node{v, l.head, nil}
	} else {
		l.tail.next = &node{v, l.tail, nil}
		l.tail = l.tail.next
	}
}
