package goroutine_pool

type node struct {
	value          *worker
	previous, next *node
}

type link struct {
	head, tail *node
}

func (l *link) getHead(fn func(w *worker) bool) *worker {
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

func (l *link) deleteHead(fn func(w *worker) bool) {
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

func (l *link) pushTail(w *worker) {
	if l.head == nil {
		l.head = &node{w, nil, nil}
	} else if l.tail == nil {
		l.tail = &node{w, l.head, nil}
	} else {
		l.tail.next = &node{w, l.tail, nil}
		l.tail = l.tail.next
	}
}
