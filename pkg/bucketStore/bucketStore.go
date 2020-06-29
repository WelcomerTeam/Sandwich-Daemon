package bucketstore

import (
	"errors"
	"sync"
	"time"

	"github.com/TheRockettek/Sandwich-Daemon/pkg/limiter"
)

// ErrNoSuchBucket is when a Bucket was requested that does not exist.
// Use CreateWaitForBucket to create a bucket if it does not exist.
var ErrNoSuchBucket = errors.New("Bucket does not exist. Use CreateWaitForBucket instead")

// BucketStore is used for managing various limiters
type BucketStore struct {
	Buckets   map[string]limiter.DurationLimiter
	BucketsMu sync.RWMutex
}

// NewBucketStore creates a new Buckets map to store different limits
func NewBucketStore() (bs *BucketStore) {
	return &BucketStore{
		Buckets:   make(map[string]limiter.DurationLimiter),
		BucketsMu: sync.RWMutex{},
	}
}

// CreateBucket will create a new bucket or overwrite
func (bs *BucketStore) CreateBucket(name string, limit int32, duration time.Duration) limiter.DurationLimiter {
	bs.BucketsMu.Lock()
	bs.Buckets[name] = limiter.NewDurationLimiter(limit, duration)
	bs.BucketsMu.Unlock()

	return bs.Buckets[name]
}

// WaitForBucket will wait for a bucket to be ready
func (bs *BucketStore) WaitForBucket(name string) (err error) {
	bucket, exists := bs.Buckets[name]

	if !exists {
		return ErrNoSuchBucket
	}
	bucket.Lock()
	return
}

// CreateWaitForBucket will create a bucket if it does not exist and then will wait
// for it.
func (bs *BucketStore) CreateWaitForBucket(name string, limit int32, duration time.Duration) (err error) {
	bs.BucketsMu.RLock()
	bucket, exists := bs.Buckets[name]
	bs.BucketsMu.RUnlock()

	if !exists {
		bucket = bs.CreateBucket(name, limit, duration)
	}
	bucket.Lock()
	return
}
