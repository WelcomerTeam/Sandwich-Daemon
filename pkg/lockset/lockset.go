package lockset

import "sync"

type void struct{}

// LockSet allows for a python-like set which allows for concurrent use
type LockSet struct {
	sync.RWMutex
	Values map[string]void `json:"values" msgpack:"values"`
}

// Contains returns a boolean if the set contains a specific value
func (ls *LockSet) Contains(val string) (contains bool) {
	ls.RLock()
	defer ls.RUnlock()

	_, contains = ls.Values[val]
	return
}

// Get returns the value of the LockSet
func (ls *LockSet) Get() (values []string) {
	ls.RLock()
	defer ls.RUnlock()

	values = make([]string, len(ls.Values))
	for key := range ls.Values {
		values = append(values, key)
	}

	return
}

// Len returns the size of the LockSet
func (ls *LockSet) Len() (count int) {
	ls.RLock()
	defer ls.RUnlock()

	return len(ls.Values)
}

// Remove removes a value from the LockSet
func (ls *LockSet) Remove(val string) (values []string, change bool) {
	ls.Lock()
	defer ls.Unlock()

	if _, ok := ls.Values[val]; ok {
		delete(ls.Values, val)
		change = true
	}

	values = make([]string, len(ls.Values))
	for key := range ls.Values {
		values = append(values, key)
	}

	return
}

// Add adds a value to the LockSet
func (ls *LockSet) Add(val string) (values []string, change bool) {
	ls.Lock()
	defer ls.Unlock()

	if _, ok := ls.Values[val]; !ok {
		ls.Values[val] = void{}
		change = true
	}

	values = make([]string, len(ls.Values))
	for key := range ls.Values {
		values = append(values, key)
	}

	return
}
