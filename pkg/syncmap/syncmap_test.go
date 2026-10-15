package syncmap

import (
	"sync"
	"testing"
)

func TestUpdateConcurrent(t *testing.T) {
	m := NewSyncMap[int, int]()
	m.Store(1, 0)
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				m.Update(1, func(value int) int { return value + 1 })
			}
		}()
	}
	wg.Wait()
	if got, _ := m.Load(1); got != 1600 {
		t.Fatalf("lost updates: got %d, want 1600", got)
	}
	if m.Update(2, func(value int) int {
		t.Fatal("callback invoked for absent key")
		return value
	}) {
		t.Fatal("Update inserted an absent key")
	}
}

func TestRangeStillAllowsMutation(t *testing.T) {
	m := NewSyncMap[int, int]()
	m.Store(1, 10)
	m.Store(2, 20)
	visited := 0
	m.Range(func(key, value int) bool {
		m.Delete(key)
		visited++
		return true
	})
	if visited != 2 || m.Count() != 0 {
		t.Fatal("snapshot iteration no longer allows mutation")
	}
}

func TestRangeReadLocked(t *testing.T) {
	m := NewSyncMap[int, int]()
	m.Store(1, 10)
	m.Store(2, 20)
	sum := 0
	m.RangeReadLocked(func(_, value int) bool {
		sum += value
		return true
	})
	if sum != 30 {
		t.Fatalf("sum = %d, want 30", sum)
	}
	visited := 0
	m.RangeReadLocked(func(_, _ int) bool {
		visited++
		return false
	})
	if visited != 1 {
		t.Fatal("early stop ignored")
	}
}

var benchmarkSum int

func BenchmarkMapIteration(b *testing.B) {
	m := NewSyncMap[int, [16]int64]()
	for i := range 1000 {
		m.Store(i, [16]int64{int64(i)})
	}
	for _, locked := range []bool{false, true} {
		name := "Snapshot"
		if locked {
			name = "ReadLocked"
		}
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				sum := 0
				visit := func(_ int, value [16]int64) bool {
					sum += int(value[0])
					return true
				}
				if locked {
					m.RangeReadLocked(visit)
				} else {
					m.Range(visit)
				}
				benchmarkSum = sum
			}
		})
	}
}
