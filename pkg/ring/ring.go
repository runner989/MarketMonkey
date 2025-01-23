package ring

import (
	"sync"
)

type Buffer[T any] struct {
	mu    sync.Mutex
	items []T
	head  int
	tail  int
	size  int
	count int
}

func NewBuffer[T any](capacity int) *Buffer[T] {
	return &Buffer[T]{
		items: make([]T, capacity),
		size:  capacity,
	}
}

func (rb *Buffer[T]) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()
	return rb.count
}

func (rb *Buffer[T]) Push(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == rb.size {
		rb.head = (rb.head + 1) % rb.size
	} else {
		rb.count++
	}
	rb.items[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.size
}

func (rb *Buffer[T]) Pop() (T, bool) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		var zero T
		return zero, false
	}
	val := rb.items[rb.head]
	var zero T
	rb.items[rb.head] = zero
	rb.head = (rb.head + 1) % rb.size
	rb.count--
	return val, true
}

func (rb *Buffer[T]) Last() T {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		var zero T
		return zero
	}
	return rb.items[rb.tail-1]
}

func (rb *Buffer[T]) SecondLast() T {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		var zero T
		return zero
	}
	return rb.items[rb.tail-2]
}

func (rb *Buffer[T]) First() T {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if rb.count == 0 {
		var zero T
		return zero
	}
	return rb.items[rb.head]
}

func (rb *Buffer[T]) GetRange(start, end int) []T {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if start < 0 || end >= rb.count {
		return nil
	}
	return rb.items[start:end]
}

func (rb *Buffer[T]) SetLast(item T) {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.items[rb.tail-1] = item
}
