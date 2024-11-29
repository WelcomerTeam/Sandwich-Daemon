package internal

import csmap "github.com/mhmtszr/concurrent-swiss-map"

// A single key to value cache
type Cache[K comparable, V any] struct {
	inner *csmap.CsMap[K, V]
	size  uint64
}

func (c *Cache[K, V]) Load(key K) (value V, ok bool) {
	if c.inner == nil {
		return
	}

	return c.inner.Load(key)
}

func (c *Cache[K, V]) Store(key K, value V) {
	if c.inner == nil {
		c.inner = csmap.Create(
			csmap.WithSize[K, V](c.size),
		)
	}

	c.inner.Store(key, value)
}

func (c *Cache[K, V]) Delete(key K) {
	if c.inner == nil {
		return
	}

	c.inner.Delete(key)
}

// Update runs a function on a value in the cache, updating the value in cache based on returned value.
func (c *Cache[K, V]) Update(key K, fn func(value V) V) (value V, ok bool) {
	if c.inner == nil {
		return
	}

	value, ok = c.inner.Load(key)
	if !ok {
		return
	}

	value = fn(value)

	c.inner.Store(key, value)

	return
}

// Range If the callback function returns true iteration will stop.
func (c *Cache[K, V]) Range(fn func(key K, value V) bool) {
	if c.inner == nil {
		return
	}

	c.inner.Range(fn)
}

func (c *Cache[K, V]) Count() int {
	if c.inner == nil {
		return 0
	}

	return c.inner.Count()
}

func (c *Cache[K, V]) Clear() {
	if c.inner == nil {
		return
	}

	c.inner.Clear()
}

func (c *Cache[K, V]) SetIfAbsent(key K, value V) {
	if c.inner == nil {
		c.inner = csmap.Create(
			csmap.WithSize[K, V](c.size),
		)
	}

	c.inner.SetIfAbsent(key, value)
}

func (c *Cache[K, V]) SetIfPresent(key K, value V) {
	if c.inner == nil {
		c.inner = csmap.Create(
			csmap.WithSize[K, V](c.size),
		)
	}

	c.inner.SetIfPresent(key, value)
}

func NewCache[K comparable, V any](size uint64) Cache[K, V] {
	return Cache[K, V]{
		size: size,
	}
}

// A 2 key to value cache
type DoubleCache[KA comparable, KB comparable, V any] struct {
	inner     Cache[KA, Cache[KB, V]]
	sizeInner uint64
}

func (c *DoubleCache[KA, KB, V]) Inner(key KA) (value Cache[KB, V], ok bool) {
	return c.inner.Load(key)
}

func (c *DoubleCache[KA, KB, V]) Load(key KA, subKey KB) (value V, ok bool) {
	if inner, ok := c.inner.Load(key); ok {
		return inner.Load(subKey)
	}

	return
}

func (c *DoubleCache[KA, KB, V]) LoadOrNew(key KA) Cache[KB, V] {
	if inner, ok := c.inner.Load(key); ok {
		return inner
	}

	inner := NewCache[KB, V](c.sizeInner)
	c.inner.Store(key, inner)

	return inner
}

func (c *DoubleCache[KA, KB, V]) Store(key KA, subKey KB, value V) {
	if inner, ok := c.inner.Load(key); ok {
		inner.Store(subKey, value)
	} else {
		inner = NewCache[KB, V](c.sizeInner)
		inner.Store(subKey, value)

		c.inner.SetIfAbsent(key, inner)
	}
}

func (c *DoubleCache[KA, KB, V]) Delete(key KA, subKey KB) {
	if inner, ok := c.inner.Load(key); ok {
		inner.Delete(subKey)
	}
}

func (c *DoubleCache[KA, KB, V]) Update(key KA, subKey KB, fn func(value V) V) (value V, ok bool) {
	if inner, ok := c.inner.Load(key); ok {
		return inner.Update(subKey, fn)
	}

	return
}

func (c *DoubleCache[KA, KB, V]) Range(fn func(key KA, value Cache[KB, V]) bool) {
	c.inner.Range(fn)
}

// Returns the total count of all values in the cache.
func (c *DoubleCache[KA, KB, V]) TotalCount() int {
	count := 0

	c.inner.Range(func(key KA, inner Cache[KB, V]) bool {
		count += inner.Count()

		return false
	})

	return count
}

// Returns the count of values in the cache for a specific key.
func (c *DoubleCache[KA, KB, V]) Count(key KA) int {
	if inner, ok := c.inner.Load(key); ok {
		return inner.Count()
	}

	return 0
}

// Clears the cache entirely.
func (c *DoubleCache[KA, KB, V]) Clear() {
	c.inner.Clear()
}

// Clears the cache for a specific key.
func (c *DoubleCache[KA, KB, V]) ClearKey(key KA) {
	c.inner.Delete(key)
}

// SetIfAbsent sets a value in the cache if it doesn't already exist.
func (c *DoubleCache[KA, KB, V]) SetIfAbsent(key KA, subKey KB, value V) {
	if inner, ok := c.inner.Load(key); ok {
		inner.SetIfAbsent(subKey, value)
	} else {
		inner = NewCache[KB, V](c.sizeInner)
		inner.SetIfAbsent(subKey, value)

		c.inner.Store(key, inner)
	}
}

// SetIfPresent sets a value in the cache if it already exists.
func (c *DoubleCache[KA, KB, V]) SetIfPresent(key KA, subKey KB, value V) {
	if inner, ok := c.inner.Load(key); ok {
		inner.SetIfPresent(subKey, value)
	} else {
		inner = NewCache[KB, V](c.sizeInner)
		inner.SetIfPresent(subKey, value)

		c.inner.Store(key, inner)
	}
}

func NewDoubleCache[KA comparable, KB comparable, V any](sizeOuter uint64, sizeInner uint64) DoubleCache[KA, KB, V] {
	return DoubleCache[KA, KB, V]{
		inner:     NewCache[KA, Cache[KB, V]](sizeOuter),
		sizeInner: sizeInner,
	}
}
