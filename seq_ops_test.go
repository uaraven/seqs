// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"math"
	"runtime"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// Accumulate used atomic.Value.CompareAndSwap, which panics when the accumulator type is not comparable.
func TestAccumulate_NonComparableAccumulator(t *testing.T) {
	inSubprocess(t, 20*time.Second, func(t *testing.T) {
		for _, parallelism := range []int{1, 4} {
			result := NewSeqFromSlice(makeSlice(100), parallelism).Accumulate([]int{}, func(a []int, b int) []int {
				return append(a, b)
			})
			assert.ElementsMatch(t, makeSlice(100), result, "parallelism=%d", parallelism)
		}

		result := NewSeqFromSlice([]string{"a", "b", "c"}, 3).Accumulate(map[string]int{}, func(a map[string]int, b string) map[string]int {
			a[b] = len(b)
			return a
		})
		assert.Equal(t, map[string]int{"a": 1, "b": 1, "c": 1}, result)
	})
}

// atomic.Value.Store panics when storing a nil interface value.
func TestAccumulate_NilInterfaceInitialValue(t *testing.T) {
	inSubprocess(t, 20*time.Second, func(t *testing.T) {
		var initial any
		for _, parallelism := range []int{1, 4} {
			result := NewSeqFromSlice(makeSlice(101), parallelism).Accumulate(initial, func(a any, b int) any {
				if a == nil {
					return b
				}
				return a.(int) + b
			})
			assert.Equal(t, 5050, result, "parallelism=%d", parallelism)
		}

		var initialErr error
		err := NewSeqFromSlice(makeSlice(3), 1).Accumulate(initialErr, func(a error, b int) error { return a })
		assert.NoError(t, err)
	})
}

// atomic.Value panics if the dynamic type of the stored value changes ("inconsistently typed value").
func TestAccumulate_AccumulatorChangesDynamicType(t *testing.T) {
	inSubprocess(t, 20*time.Second, func(t *testing.T) {
		result := NewSeqFromSlice(makeSlice(101), 1).Accumulate(any(0), func(a any, b int) any {
			switch v := a.(type) {
			case int:
				return int64(v + b)
			default:
				return v.(int64) + int64(b)
			}
		})
		assert.Equal(t, int64(5050), result)
	})
}

// With a NaN accumulator CompareAndSwap compares NaN == NaN, which is never true, so the CAS loop spins forever.
func TestAccumulate_NaNAccumulatorDoesNotLivelock(t *testing.T) {
	inSubprocess(t, 5*time.Second, func(t *testing.T) {
		for _, parallelism := range []int{1, 4} {
			result := NewSeqFromSlice(makeSlice(10), parallelism).Accumulate(math.NaN(), func(a float64, b int) float64 {
				return a + float64(b)
			})
			assert.True(t, math.IsNaN(result), "parallelism=%d", parallelism)
		}
	})
}

// When the CAS fails, the reducer used to be run again for the same element, so reducers with side effects
// were executed more than once per element and the call count exceeded the number of elements.
func TestAccumulate_ReducerCalledOncePerElement(t *testing.T) {
	const elements = 2000
	for _, parallelism := range []int{2, 4, 8, 32} {
		var calls atomic.Int64
		result := NewSeqFromSlice(makeSlice(elements), parallelism).Accumulate(int64(0), func(a int64, b int) int64 {
			calls.Add(1)
			runtime.Gosched() // widen the window so concurrent workers conflict
			return a + int64(b)
		})
		assert.EqualValues(t, elements*(elements-1)/2, result, "parallelism=%d", parallelism)
		assert.EqualValues(t, elements, calls.Load(), "reducer must be called exactly once per element, parallelism=%d", parallelism)
	}
}

// The reducer must never run concurrently with itself, otherwise reducers that mutate the accumulator are unsafe.
func TestAccumulate_ReducerIsNotRunConcurrently(t *testing.T) {
	var running, overlaps atomic.Int64
	NewSeqFromSlice(makeSlice(2000), 8).Accumulate(0, func(a int, b int) int {
		if running.Add(1) > 1 {
			overlaps.Add(1)
		}
		runtime.Gosched()
		running.Add(-1)
		return a + b
	})
	assert.EqualValues(t, 0, overlaps.Load(), "reducer calls overlapped")
}

// recovered runs f and returns the value f panicked with (nil if it did not panic).
func recovered(f func()) (p any) {
	defer func() { p = recover() }()
	f()
	return nil
}

// A panic in the reducer happens on a worker goroutine, where the caller cannot recover it. Accumulate must
// re-panic on the caller's goroutine with the original value, and must not deadlock or kill the process.
func TestAccumulate_ReducerPanicIsPropagatedToCaller(t *testing.T) {
	for _, parallelism := range []int{1, 2, 8} {
		runWithTimeout(t, func() {
			p := recovered(func() {
				NewSeqFromSlice(makeSlice(1000), parallelism).Accumulate(0, func(a, b int) int {
					if b == 500 {
						panic("boom")
					}
					return a + b
				})
			})
			assert.Equal(t, "boom", p, "parallelism=%d", parallelism)
		})
	}
}

// The panic value must be passed through unchanged, whatever its type.
func TestAccumulate_ReducerPanicValueIsPreserved(t *testing.T) {
	err := assert.AnError
	p := recovered(func() {
		NewSeqFromSlice(makeSlice(10), 1).Accumulate(0, func(a, b int) int { panic(err) })
	})
	assert.Same(t, err, p)
}

// Every worker may panic at once. Accumulate must still return control to the caller, with a panic.
func TestAccumulate_AllWorkersPanic(t *testing.T) {
	for _, parallelism := range []int{2, 8, 32} {
		runWithTimeout(t, func() {
			p := recovered(func() {
				NewSeqFromSlice(makeSlice(1000), parallelism).Accumulate(0, func(a, b int) int { panic("always") })
			})
			assert.Equal(t, "always", p, "parallelism=%d", parallelism)
		})
	}
}

// The mutex must be released when the reducer panics, otherwise the other workers block forever on Lock and
// wg.Wait never returns.
func TestAccumulate_ReducerPanicDoesNotLeaveMutexLocked(t *testing.T) {
	for _, parallelism := range []int{2, 8} {
		var calls atomic.Int64
		runWithTimeout(t, func() {
			recovered(func() {
				NewSeqFromSlice(makeSlice(1000), parallelism).Accumulate(0, func(a, b int) int {
					calls.Add(1)
					if b == 0 {
						panic("first element")
					}
					return a + b
				})
			})
		})
		// Surviving workers keep consuming after one worker died, so the reducer ran for more than the failing call.
		assert.Greater(t, calls.Load(), int64(1), "parallelism=%d", parallelism)
	}
}

// A successful Accumulate after a failed one must not be affected by it (no shared or leaked state).
func TestAccumulate_UsableAfterPanic(t *testing.T) {
	recovered(func() {
		NewSeqFromSlice(makeSlice(10), 4).Accumulate(0, func(a, b int) int { panic("x") })
	})
	assert.Equal(t, 45, NewSeqFromSlice(makeSlice(10), 4).Accumulate(0, func(a, b int) int { return a + b }))
}

// Count is built on Accumulate and has no reducer of its own that can panic, but a panicking upstream stage that runs
// on the Accumulate worker (sequential Map) must surface to the caller too.
func TestAccumulate_UpstreamPanicIsPropagated(t *testing.T) {
	runWithTimeout(t, func() {
		p := recovered(func() {
			NewSeqFromSlice(makeSlice(10), 1).Map(func(v int) int {
				if v == 3 {
					panic("mapper")
				}
				return v
			}).Accumulate(0, func(a, b int) int { return a + b })
		})
		assert.Equal(t, "mapper", p)
	})
}

func TestAccumulate_Basics(t *testing.T) {
	for _, parallelism := range []int{1, 4, 16} {
		assert.Equal(t, 15, NewSeqFromSlice([]int{1, 2, 3, 4, 5}, parallelism).Accumulate(0, func(a, b int) int { return a + b }))
		assert.Equal(t, 7, NewSeqFromSlice([]int{}, parallelism).Accumulate(7, func(a, b int) int { return a + b }))
	}
	assert.Equal(t, 7, Seq[int]{}.Accumulate(7, func(a, b int) int { return a + b }))
}

// Count is built on Accumulate. These guard its result under heavy contention.
func TestCount_ExactUnderContention(t *testing.T) {
	for _, parallelism := range []int{1, 2, 8, 32} {
		runWithTimeout(t, func() {
			assert.EqualValues(t, 100000, NewSeqFromSlice(makeSlice(100000), parallelism).Count(), "parallelism=%d", parallelism)
		})
	}
	assert.EqualValues(t, 0, NewSeqFromSlice([]int{}, 4).Count())
	assert.EqualValues(t, 0, Seq[int]{}.Count())
}

// Count must not invoke the user-visible pipeline more than once per element.
func TestCount_EvaluatesEachElementOnce(t *testing.T) {
	var seen atomic.Int64
	n := NewSeqFromSlice(makeSlice(5000), 8).Peek(func(int) { seen.Add(1) }).Count()
	assert.EqualValues(t, 5000, n)
	assert.EqualValues(t, 5000, seen.Load())
}
