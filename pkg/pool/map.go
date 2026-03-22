package pool

import (
	"sync"
)

type MapPool[K comparable, V any] struct {
	size int
	pool sync.Pool
}

func NewMapPool[K comparable, V any](size int) *MapPool[K, V] {
	return &MapPool[K, V]{
		size: size,
		pool: sync.Pool{
			New: func() any { return make(map[K]V, size) },
		},
	}
}

func (mp *MapPool[K, V]) Get() map[K]V {
	return mp.pool.Get().(map[K]V)
}

func (mp *MapPool[K, V]) Put(m map[K]V) {
	if len(m) > mp.size {
		return // Discard if grown
	}

	clear(m)
	mp.pool.Put(m)
}
