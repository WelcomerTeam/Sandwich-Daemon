package syncmap

import "sync"

// Map is a type-safe wrapper around sync.Map
type Map[K comparable, V any] struct {
	m sync.Map
}

// Store stores the value for the key
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Load loads the value for the key
func (m *Map[K, V]) Load(key K) (V, bool) {
	value, ok := m.m.Load(key)
	if !ok {
		var zero V

		return zero, false
	}

	return value.(V), true
}

// Delete deletes the value for the key
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// LoadAndDelete loads and deletes the value for the key
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	value, ok := m.m.LoadAndDelete(key)
	if !ok {
		var zero V

		return zero, false
	}

	return value.(V), true
}

// LoadOrStore loads the value for the key if it exists, otherwise stores and returns the given value
func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	actual, loaded := m.m.LoadOrStore(key, value)

	return actual.(V), loaded
}

// Range calls f for each key-value pair in the map
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(key, value interface{}) bool {
		return f(key.(K), value.(V))
	})
}
