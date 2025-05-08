package atomic

import "sync/atomic"

// Int32 is a wrapper around atomic.Int32
type Int32 struct {
	value atomic.Int32
}

// NewInt32 creates a new Int32 with the given initial value
func NewInt32(initial int32) *Int32 {
	i := &Int32{}
	i.Store(initial)
	return i
}

// Store stores the value
func (i *Int32) Store(value int32) {
	i.value.Store(value)
}

// Load loads the value
func (i *Int32) Load() int32 {
	return i.value.Load()
}

// Add adds delta to the value and returns the new value
func (i *Int32) Add(delta int32) int32 {
	return i.value.Add(delta)
}

// Swap swaps the value and returns the old value
func (i *Int32) Swap(new int32) int32 {
	return i.value.Swap(new)
}

// CompareAndSwap compares and swaps the value
func (i *Int32) CompareAndSwap(old, new int32) bool {
	return i.value.CompareAndSwap(old, new)
}
