package syncmap

import (
	"sync"
)

// Map is a type-safe wrapper around map
type Map[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

func NewSyncMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		m:  make(map[K]V),
		mu: sync.RWMutex{},
	}
}

// Store stores the value for the key
func (m *Map[K, V]) Store(key K, value V) {
	m.mu.Lock()
	if m.m == nil {
		m.m = make(map[K]V)
	}
	m.m[key] = value
	m.mu.Unlock()
}

// Load loads the value for the key
func (m *Map[K, V]) Load(key K) (V, bool) {
	m.mu.RLock()
	value, ok := m.m[key]
	m.mu.RUnlock()
	if !ok {
		var zero V

		return zero, false
	}

	return value, true
}

// Delete deletes the value for the key
func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	delete(m.m, key)
	m.mu.Unlock()
}

// LoadAndDelete loads and deletes the value for the key
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	m.mu.Lock()
	value, ok := m.m[key]
	if ok {
		delete(m.m, key)
	}
	m.mu.Unlock()

	if !ok {
		var zero V

		return zero, false
	}

	return value, true
}

// LoadOrStore loads the value for the key if it exists, otherwise stores and returns the given value
func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	m.mu.Lock()
	if m.m == nil {
		m.m = make(map[K]V)
	}

	actual, loaded := m.m[key]
	if !loaded {
		m.m[key] = value
		actual = value
	}
	m.mu.Unlock()

	return actual, loaded
}

// Range calls f for each key-value pair in the map
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	type kv struct {
		k K
		v V
	}

	m.mu.RLock()
	items := make([]kv, 0, len(m.m))
	for k, v := range m.m {
		items = append(items, kv{k: k, v: v})
	}
	m.mu.RUnlock()

	for _, item := range items {
		if !f(item.k, item.v) {
			return
		}
	}
}

// Count returns the number of items in the map
// This is an O(1) operation using len(map)
func (m *Map[K, V]) Count() int {
	m.mu.RLock()
	count := len(m.m)
	m.mu.RUnlock()

	return count
}
