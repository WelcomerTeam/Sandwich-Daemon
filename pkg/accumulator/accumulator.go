package accumulator

import (
	"context"
	"sync"
	"time"
)

// Accumulator stores data on an interval and allows you to access it
type Accumulator struct {
	ctx context.Context

	sync.RWMutex
	Label   string // used to identify the owner of an accumulator
	Samples []*Sample
	acc     int64

	// Samples to store before being discarded
	storedSamples int

	// Time between sampling from the accumulator.
	// 600 samples with an interval of 1 second will provide a 10 minute history.
	// 5760 with an interval of 15 seconds will provide a 1 day history.
	interval time.Duration
}

// Sample contains the time the sample was made and its value
type Sample struct {
	Value    int64     `json:"value"`
	StoredAt time.Time `json:"stored_at"`
}

// Increment increments the accumulator by 1
func (ac *Accumulator) Increment() {
	ac.IncrementBy(1)
}

// IncrementBy increments the accumulator by a specified number
func (ac *Accumulator) IncrementBy(acc int64) {
	ac.Lock()
	defer ac.Unlock()
	ac.acc += acc
}

// GetAllSamples returns all samples from the accumulator
func (ac *Accumulator) GetAllSamples() *SampleGroup {
	return &SampleGroup{ac.Label, ac.Samples}
}

// GetLastSamples returns the last N samples from the accumulator
func (ac *Accumulator) GetLastSamples(n int) *SampleGroup {
	index := len(ac.Samples) - n
	if index < 0 {
		index = 0
	}

	ac.RLock()
	defer ac.RUnlock()
	return &SampleGroup{ac.Label, ac.Samples[index:]}
}

// GetSamplesSince returns the last N samples from the accumulator in the specified time
func (ac *Accumulator) GetSamplesSince(t time.Time) *SampleGroup {
	ac.RLock()
	defer ac.RUnlock()
	for index := range ac.Samples {
		if ac.Samples[len(ac.Samples)-index].StoredAt.After(t) {
			return &SampleGroup{ac.Label, ac.Samples[index:]}
		}
	}
	return &SampleGroup{ac.Label, ac.Samples}
}

// SampleGroup holds a group of samples
type SampleGroup struct {
	Label   string
	Samples []*Sample
}

// Sum returns the sum of all samples in a samplegroup object
func (sg *SampleGroup) Sum() int64 {
	acc := int64(0)
	for _, sample := range sg.Samples {
		acc += sample.Value
	}
	return acc
}

// Avg returns the average of all samples in a samplegroup object
func (sg *SampleGroup) Avg() float64 {
	return float64(sg.Sum()) / float64(len(sg.Samples))
}

// Since returns a samplegroup with all samples after a specified time
func (sg *SampleGroup) Since(t time.Time) *SampleGroup {
	for index := range sg.Samples {
		if sg.Samples[len(sg.Samples)-index].StoredAt.After(t) {
			return &SampleGroup{"", sg.Samples[index:]}
		}
	}
	return &SampleGroup{"", sg.Samples}
}

// RunOnce allows you to manually call the accumulator task in the event you already have a task running every interval
func (ac *Accumulator) RunOnce(t time.Time) {
	ac.Lock()
	ac.Samples = append(ac.Samples, &Sample{
		ac.acc,
		t,
	})

	// Reset accumulator
	ac.acc = 0

	// If we surpass the stored samples number, remove old samples
	if len(ac.Samples) > ac.storedSamples {
		ac.Samples = ac.Samples[len(ac.Samples)-ac.storedSamples:]
	}
	ac.Unlock()
}

// Run starts the accumulator which will process the accumulator and store it appropriately.
func (ac *Accumulator) Run() {
	t := time.NewTicker(ac.interval)
	for {
		select {
		case <-ac.ctx.Done():
			return
		case <-t.C:
		}

		ac.RunOnce(time.Now().UTC())
	}
}

// NewAccumulator creates an accumulator. This does not automatically call Run
func NewAccumulator(ctx context.Context, storedSamples int, interval time.Duration) *Accumulator {
	acc := &Accumulator{
		Samples:       make([]*Sample, 0, storedSamples),
		acc:           int64(0),
		ctx:           ctx,
		storedSamples: storedSamples,
		interval:      interval,
	}

	return acc
}
