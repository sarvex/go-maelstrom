package crdt

import "sync"

type Accumulator[T any, V any] struct {
	mu          sync.RWMutex
	value       T
	accumulator func(acc T, new V) T
	cp          func(cur T) T
}

func CreateAccumulator[T any, V any](initVal T, acc func(acc T, new V) T, cp func(cur T) T) *Accumulator[T, V] {
	return &Accumulator[T, V]{
		value:       initVal,
		accumulator: acc,
		cp:          cp,
	}
}

func (acc *Accumulator[T, V]) Add(value V) {
	acc.mu.Lock()
	acc.value = acc.accumulator(acc.value, value)
	acc.mu.Unlock()
}

func (acc *Accumulator[T, V]) Set(other T) {
	acc.mu.Lock()
	acc.value = other
	acc.mu.Unlock()
}

func (acc *Accumulator[T, V]) Get() T {
	acc.mu.RLock()
	cp := acc.cp(acc.value)
	acc.mu.RUnlock()
	return cp
}
