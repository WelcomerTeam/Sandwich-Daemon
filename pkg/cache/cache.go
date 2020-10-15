package cache

import (
	"math/rand"
	"reflect"
	"time"
)

// Cache allows to temporarily store values and will retrieve new ones
// similar to a Pool if it does not exist
type Cache struct {
	store map[int64]*StorePair

	ttl time.Duration

	// New specifies a function to retrieve a new value when Get
	// would otherwise return nil.
	New func(key int64) interface{}
}

// StorePair represents a single entry in the store
type StorePair struct {
	value      interface{}
	expiration time.Time
}

// Get retrieves from the cache and if it does not exist it will call New
func (c *Cache) Get(key int64) interface{} {
	pair, ok := c.store[key]
	if !ok {
		val := c.New(key)
		c.store[key] = &StorePair{val, time.Now().Add(c.ttl)}
		return val
	}
	return pair.value
}

// Run will clear old values in the store. It will choose a random key from
// store every second and if it has expired, it will remove it and repeat until
// it finds a key that has not expired.
func (c *Cache) Run() {
	t := time.NewTicker(time.Second)
	for {
		<-t.C
		now := time.Now()
		for {
			keys := reflect.ValueOf(c.store).MapKeys()
			key := keys[rand.Intn(len(c.store))].Int()
			pair := c.store[key]
			if pair.expiration.Before(now) {
				delete(c.store, key)
			} else {
				break
			}
		}
	}
}

// Empty removes a key from the store
func (c *Cache) Empty(key int64) {
	delete(c.store, key)
}
