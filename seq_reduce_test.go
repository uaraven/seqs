// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var reduceParallelism = []int{1, 2, 3, 4, 8, 16, 64}

func sum(a, b int) int { return a + b }

// Reduce must give the same result as a plain loop for every combination of source size and parallelism, including
// the sizes where some (or all) workers get no elements.
func TestReduce_ResultMatchesSequentialLoop(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 7, 64, 1000, 20000} {
		expected := 0
		for i := range size {
			expected += i
		}
		for _, parallelism := range reduceParallelism {
			runWithTimeout(t, func() {
				got := NewSeqFromSlice(makeSlice(size), parallelism).Reduce(sum)
				assert.Equal(t, expected, got, "size=%d parallelism=%d", size, parallelism)
			})
		}
	}
}

// Every element must be taken into account exactly once: none lost and none counted twice.
func TestReduce_EveryElementUsedExactlyOnce(t *testing.T) {
	const size = 300
	input := make([][]int, size)
	for i := range input {
		input[i] = []int{i}
	}
	concat := func(a, b []int) []int {
		r := make([]int, 0, len(a)+len(b))
		return append(append(r, a...), b...)
	}
	for _, parallelism := range reduceParallelism {
		for range 5 {
			runWithTimeout(t, func() {
				got := NewSeqFromSlice(input, parallelism).Reduce(concat)
				assert.ElementsMatch(t, makeSlice(size), got, "parallelism=%d", parallelism)
			})
		}
	}
}

// Merging n elements takes exactly n-1 reducer calls no matter how the elements are distributed between workers.
func TestReduce_ReducerCalledNMinusOneTimes(t *testing.T) {
	for _, size := range []int{1, 2, 5, 100, 5000} {
		for _, parallelism := range reduceParallelism {
			var calls atomic.Int64
			runWithTimeout(t, func() {
				NewSeqFromSlice(makeSlice(size), parallelism).Reduce(func(a, b int) int {
					calls.Add(1)
					return a + b
				})
			})
			assert.EqualValues(t, size-1, calls.Load(), "size=%d parallelism=%d", size, parallelism)
		}
	}
}

// Zero, one and a missing source must not block or panic.
func TestReduce_DegenerateInputs(t *testing.T) {
	for _, parallelism := range reduceParallelism {
		runWithTimeout(t, func() {
			assert.Equal(t, 0, NewSeqFromSlice([]int{}, parallelism).Reduce(sum), "empty, parallelism=%d", parallelism)
			assert.Equal(t, 42, NewSeqFromSlice([]int{42}, parallelism).Reduce(sum), "single, parallelism=%d", parallelism)
			assert.Equal(t, "", NewSeqFromSlice([]string{}, parallelism).Reduce(func(a, b string) string { return a + b }))
		})
	}
	runWithTimeout(t, func() {
		assert.Equal(t, 0, Seq[int]{}.Reduce(sum))
		assert.Equal(t, 0, EmptySeq[int]().Reduce(sum))
	})
}

// A sequential Seq must reduce strictly from left to right, so non-commutative reducers keep their meaning.
func TestReduce_SequentialKeepsOrder(t *testing.T) {
	got := SeqOf("a", "b", "c", "d", "e").Reduce(func(a, b string) string { return a + b })
	assert.Equal(t, "abcde", got)
	assert.Equal(t, -13, SeqOf(1, 2, 3, 4, 5).Reduce(func(a, b int) int { return a - b }))
}

// min/max are associative and commutative, so the answer must not depend on how work is split.
func TestReduce_MinMaxParallel(t *testing.T) {
	data := make([]int, 5000)
	for i := range data {
		data[i] = (i * 7919) % 10007
	}
	minV, maxV := data[0], data[0]
	for _, v := range data {
		minV, maxV = min(minV, v), max(maxV, v)
	}
	for _, parallelism := range reduceParallelism {
		runWithTimeout(t, func() {
			assert.Equal(t, minV, NewSeqFromSlice(data, parallelism).Reduce(func(a, b int) int { return min(a, b) }))
			assert.Equal(t, maxV, NewSeqFromSlice(data, parallelism).Reduce(func(a, b int) int { return max(a, b) }))
		})
	}
}

// Many Reduce calls on independent sequences at once must not interfere with each other (run with -race).
func TestReduce_ConcurrentCallsOnIndependentSeqs(t *testing.T) {
	const callers = 32
	var wg sync.WaitGroup
	var wrong atomic.Int64
	runWithTimeout(t, func() {
		for c := range callers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				size := 1000 + c
				got := NewSeqFromSlice(makeSlice(size), 4).Reduce(sum)
				if got != size*(size-1)/2 {
					wrong.Add(1)
				}
			}()
		}
		wg.Wait()
	})
	assert.EqualValues(t, 0, wrong.Load())
}

// The source is fed from a channel by a slow producer, so workers wait on the source at different times.
func TestReduce_ChannelSourceSlowProducer(t *testing.T) {
	for _, parallelism := range []int{1, 4, 16} {
		const size = 500
		ch := make(chan int)
		go func() {
			defer close(ch)
			for i := range size {
				ch <- i
				if i%50 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}()
		runWithTimeout(t, func() {
			got := NewParallelSeq(NewChannelProducer(ch), parallelism).Reduce(sum)
			assert.Equal(t, size*(size-1)/2, got, "parallelism=%d", parallelism)
		})
	}
}

// Reduce on top of other lazy stages: the pipeline must be consumed exactly once.
func TestReduce_AfterPipelineStages(t *testing.T) {
	for _, parallelism := range reduceParallelism {
		var peeked atomic.Int64
		runWithTimeout(t, func() {
			got := NewSeqFromSlice(makeSlice(1000), parallelism).
				Peek(func(int) { peeked.Add(1) }).
				Filter(func(v int) bool { return v%2 == 0 }).
				Map(func(v int) int { return v * 2 }).
				Reduce(sum)
			expected := 0
			for i := 0; i < 1000; i += 2 {
				expected += i * 2
			}
			assert.Equal(t, expected, got, "parallelism=%d", parallelism)
		})
		assert.EqualValues(t, 1000, peeked.Load(), "parallelism=%d", parallelism)
	}
}

// Reducing the same Seq twice: the first call consumes it, the second must return the zero value without hanging.
func TestReduce_SecondCallOnConsumedSeq(t *testing.T) {
	for _, parallelism := range []int{1, 4} {
		seq := NewSeqFromSlice(makeSlice(100), parallelism)
		runWithTimeout(t, func() {
			assert.Equal(t, 4950, seq.Reduce(sum))
			assert.Equal(t, 0, seq.Reduce(sum))
		})
	}
}

// The reducer runs on several goroutines, but never for the same partial result at the same time. Verify by
// using a reducer that is only safe if its accumulator argument is not shared between calls.
func TestReduce_AccumulatorNotSharedBetweenCalls(t *testing.T) {
	type box struct{ n int }
	input := make([]*box, 3000)
	for i := range input {
		input[i] = &box{n: 1}
	}
	for _, parallelism := range reduceParallelism {
		for _, b := range input {
			b.n = 1
		}
		runWithTimeout(t, func() {
			// mutates and returns the left operand; a partial result handed to two calls would race or double count
			got := NewSeqFromSlice(input, parallelism).Reduce(func(a, b *box) *box {
				runtime.Gosched()
				a.n += b.n
				return a
			})
			assert.Equal(t, len(input), got.n, "parallelism=%d", parallelism)
		})
	}
}
