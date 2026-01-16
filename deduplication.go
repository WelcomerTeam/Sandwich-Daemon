package sandwich

import (
	"context"
	"sync"
	"time"
)

// DedupeProvider is an interface for deduplication operations.
// It provides a method to check if a key is already being processed.
// If the key has been/already being processed, it returns false.
type DedupeProvider interface {
	Deduplicate(ctx context.Context, key string, ttl time.Duration) bool
	Release(ctx context.Context, key string)
}

// noopDedupeProvider is a no-operation implementation of DedupeProvider.
// It always returns true, indicating that the key can be processed.
// This is useful for cases where deduplication is not needed.
type noopDedupeProvider struct{}

func NewNoopDedupeProvider() *noopDedupeProvider {
	return &noopDedupeProvider{}
}

func (n *noopDedupeProvider) Deduplicate(ctx context.Context, key string, ttl time.Duration) bool {
	return true
}

func (n *noopDedupeProvider) Release(ctx context.Context, key string) {
	// No operation needed for release in noop provider
}

// inMemoryDedupeProvider is an in-memory implementation of DedupeProvider.
// It uses a map to track keys and their expiration times.
// It allows deduplication of operations based on a key and a time-to-live (TTL).
// It also includes a cleanup mechanism to remove expired keys periodically.
// The cleanup runs every minute to ensure that expired keys are removed.
type inMemoryDedupeProvider struct {
	keys map[string]time.Time
	mu   sync.RWMutex
}

func NewInMemoryDedupeProvider() *inMemoryDedupeProvider {
	p := &inMemoryDedupeProvider{
		keys: make(map[string]time.Time),
	}

	go func() {
		// Cleanup every 15 seconds instead of 1 minute to prevent unbounded growth under high load
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			p.Cleanup() // Periodically clean up expired keys
		}
	}()

	return p
}

func (d *inMemoryDedupeProvider) Deduplicate(ctx context.Context, key string, ttl time.Duration) bool {
	now := time.Now()
	expiration := now.Add(ttl)

	d.mu.Lock()
	existingTime, exists := d.keys[key]

	// Check if key exists and is still valid
	if exists && existingTime.After(now) {
		d.mu.Unlock()
		return false // Key is already being processed
	}

	// Set the key atomically within the same lock
	d.keys[key] = expiration
	d.mu.Unlock()

	return true // Key is not being processed, can proceed
}

func (d *inMemoryDedupeProvider) Release(_ context.Context, key string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	delete(d.keys, key)
}

func (d *inMemoryDedupeProvider) Cleanup() {
	now := time.Now()

	d.mu.Lock()
	for key, expiration := range d.keys {
		if expiration.Before(now) {
			delete(d.keys, key) // Remove expired keys
		}
	}
	d.mu.Unlock()
}
