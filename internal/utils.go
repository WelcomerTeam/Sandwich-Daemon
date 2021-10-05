package internal

import (
	"context"
	"encoding/hex"
	"hash"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/atomic"
)

const (
	daySeconds    = 86400
	hourSeconds   = 3600
	minuteSeconds = 60
)

type void struct{}

type CtxGroup struct {
	context context.Context
	cancel  func()
}

func replaceIfEmpty(v string, s string) string {
	if v == "" {
		return s
	}

	return v
}

func returnError(err error) string {
	if err != nil {
		return err.Error()
	}

	return ""
}

// quickHash returns hash from method and input.
func quickHash(hashMethod hash.Hash, text string) (result string, err error) {
	hashMethod.Reset()

	if _, err := hashMethod.Write([]byte(text)); err != nil {
		return "", err
	}

	return hex.EncodeToString(hashMethod.Sum(nil)), nil
}

// returnRange converts a string like 0-4,6-7 to [0,1,2,3,4,6,7].
func returnRange(_range string, max int) (result []int) {
	for _, split := range strings.Split(_range, ",") {
		ranges := strings.Split(split, "-")
		if low, err := strconv.Atoi(ranges[0]); err == nil {
			if hi, err := strconv.Atoi(ranges[len(ranges)-1]); err == nil {
				for i := low; i < hi+1; i++ {
					if 0 <= i && i < max {
						result = append(result, i)
					}
				}
			}
		}
	}

	return result
}

// webhookTime returns a formatted time.Time as a time accepted by webhooks.
func webhookTime(_time time.Time) string {
	return _time.Format("2006-01-02T15:04:05Z")
}

// Simple orchestrator to close long running tasks
// and wait for them to acknowledge completion.
type DeadSignal struct {
	sync.Mutex
	waiting sync.WaitGroup

	alreadyClosed atomic.Bool
	dead          chan void
}

func (ds *DeadSignal) init() {
	ds.Lock()
	if ds.dead == nil {
		ds.alreadyClosed = *atomic.NewBool(false)
		ds.dead = make(chan void, 1)
		ds.waiting = sync.WaitGroup{}
	}
	ds.Unlock()
}

// Returns the dead channel.
func (ds *DeadSignal) Dead() chan void {
	ds.init()

	ds.Lock()
	defer ds.Unlock()

	return ds.dead
}

// Signifies the goroutine has started.
// When calling open, done should be called on end.
func (ds *DeadSignal) Started() {
	ds.init()
	ds.waiting.Add(1)
}

// Signifies the goroutine is done.
func (ds *DeadSignal) Done() {
	ds.init()
	ds.waiting.Done()
}

// Close closes the dead channel and
// waits for other goroutines waiting on Dead() to call Done().
// When Close returns, it is designed that any goroutines will no
// longer be using it.
func (ds *DeadSignal) Close(t string) {
	ds.init()

	ds.Lock()
	if !ds.alreadyClosed.Load() {
		close(ds.dead)
		ds.alreadyClosed.Store(true)
	}
	ds.Unlock()

	ds.waiting.Wait()
}

// Revive makes a closed DeadSignal create
// a new dead channel to allow for it to be reused. You should not
// revive on a closed channel if it is still being actively used.
func (ds *DeadSignal) Revive() {
	ds.init()

	ds.Lock()
	ds.dead = make(chan void, 1)
	ds.alreadyClosed.Store(false)
	ds.Unlock()
}

// Similar to Close however does not wait for goroutines to finish.
// Both should not be ran.
func (ds *DeadSignal) Kill() {
	ds.init()

	ds.Lock()
	defer ds.Unlock()

	close(ds.dead)
}
