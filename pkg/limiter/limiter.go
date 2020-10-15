package limiter

import (
	"fmt"
	"sync/atomic"
	"time"
)

// ConcurrencyLimiter object
type ConcurrencyLimiter struct {
	name       string
	limit      int
	tickets    chan int
	inProgress int32
}

// NewConcurrencyLimiter allocates a new ConcurrencyLimiter. This is useful
// for limiting the amount of functions running at once and is used to
// only allow a specific number of sessions to start at once.
func NewConcurrencyLimiter(name string, limit int) *ConcurrencyLimiter {
	c := &ConcurrencyLimiter{
		name:    name,
		limit:   limit,
		tickets: make(chan int, limit),
	}
	for i := 0; i < c.limit; i++ {
		c.tickets <- i
	}
	return c
}

// Wait waits for a free ticket in the queue. Functions that call wait
// must defer FreeTicket with thee ticket id
func (c *ConcurrencyLimiter) Wait() (ticket int) {
	ticket = <-c.tickets
	atomic.AddInt32(&c.inProgress, 1)
	return ticket
}

// FreeTicket adds the ticket back into the queue.
func (c *ConcurrencyLimiter) FreeTicket(ticket int) {
	c.tickets <- ticket
	atomic.AddInt32(&c.inProgress, -1)
	return
}

// InProgress returns how many tickets are being used
func (c *ConcurrencyLimiter) InProgress() int32 {
	return atomic.LoadInt32(&c.inProgress)
}

// DurationLimiter represents something that will wait until the ratelimit
// has cleared
type DurationLimiter struct {
	name     string
	limit    *int32
	duration *int64

	resetsAt  *int64
	available *int32
}

// NewDurationLimiter creates a DurationLimiter. This is useful for allowing
// a specific operation to run only X amount of times in a duration of Y.
func NewDurationLimiter(name string, limit int32, duration time.Duration) (bs *DurationLimiter) {
	nanos := duration.Nanoseconds()
	bs = &DurationLimiter{
		name:     name,
		limit:    &limit,
		duration: &nanos,

		resetsAt:  new(int64),
		available: new(int32),
	}
	return bs
}

// Lock waits until there is an available slot in the Limiter
func (l *DurationLimiter) Lock() {
	now := time.Now().UnixNano()

	// If we have surpassed the resetAt, then make a new resetAt and free
	// up available
	if atomic.LoadInt64(l.resetsAt) <= now {
		atomic.StoreInt64(l.resetsAt, now+atomic.LoadInt64(l.duration))
		atomic.StoreInt32(l.available, atomic.LoadInt32(l.limit))
	}

	if atomic.LoadInt32(l.available) <= 0 {
		// This on its own can create a race condition if 2 routines are
		// waiting simultaneously. In order to not make this occur, we
		// must call the lock again to make sure.
		sleepDuration := time.Duration(atomic.LoadInt64(l.resetsAt) - now)
		println(fmt.Sprintf("%s is being ratelimited! Waiting %dms", l.name, sleepDuration.Milliseconds()))
		time.Sleep(sleepDuration)
		l.Lock()
		return
	}

	atomic.AddInt32(l.available, -1)
	return
}

// Reset resets the resetsAt
func (l *DurationLimiter) Reset() {
	now := time.Now().UnixNano()
	atomic.StoreInt64(l.resetsAt, now+atomic.LoadInt64(l.duration))
}
