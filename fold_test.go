// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

var foldParallelism = []int{1, 2, 3, 4, 8, 16, 64}

func foldSum(src Seq[int]) int {
	return foldOf(src, func() int { return 0 }, func(a, b int) int { return a + b }, func(a, b int) int { return a + b })
}

// Fold must agree with a plain loop and with Accumulate for every size/parallelism, including sizes where some or
// all workers get no elements.
func TestFold_MatchesLoopAndAccumulate(t *testing.T) {
	for _, size := range []int{0, 1, 2, 3, 7, 64, 1000, 20000} {
		expected := 0
		for i := range size {
			expected += i
		}
		for _, p := range foldParallelism {
			runWithTimeout(t, func() {
				data := makeSlice(size)
				assert.Equal(t, expected, foldSum(NewSeqFromSlice(data, p)), "size=%d p=%d", size, p)
				acc := NewSeqFromSlice(data, p).Accumulate(0, func(a, b int) int { return a + b })
				assert.Equal(t, acc, foldSum(NewSeqFromSlice(data, p)), "size=%d p=%d", size, p)
			})
		}
	}
}

// zero() must be the starting accumulator of every worker. A zero that is not Go's zero value exposes code that
// starts from `var acc R` instead.
func TestFold_UsesZeroFunctionAsStartingAccumulator(t *testing.T) {
	for _, p := range foldParallelism {
		got := foldOf(NewSeqFromSlice(makeSlice(10), p), // 1*2*...: product of (v+1) = 10!
			func() int { return 1 },
			func(a, b int) int { return a * (b + 1) },
			func(a, b int) int { return a * b })
		assert.Equal(t, 3628800, got, "p=%d", p)
	}
}

// A zero() returning a fresh reference value gives each worker its own accumulator; a nil map would panic on write.
func TestFold_ReferenceTypeAccumulator(t *testing.T) {
	words := []string{"a", "b", "a", "c", "b", "a"}
	for _, p := range foldParallelism {
		got := foldOf(NewSeqFromSlice(words, p),
			func() map[string]int { return map[string]int{} },
			func(m map[string]int, w string) map[string]int { m[w]++; return m },
			func(a, b map[string]int) map[string]int {
				for k, v := range b {
					a[k] += v
				}
				return a
			})
		assert.Equal(t, map[string]int{"a": 3, "b": 2, "c": 1}, got, "p=%d", p)
	}
}

// R may differ from T.
func TestFold_DifferentAccumulatorType(t *testing.T) {
	for _, p := range foldParallelism {
		got := foldOf(NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd"}, p),
			func() int { return 0 },
			func(n int, s string) int { return n + len(s) },
			func(a, b int) int { return a + b })
		assert.Equal(t, 10, got, "p=%d", p)
	}
}

// Empty and zero-value sequences return zero() and never call reducer.
func TestFold_EmptyAndZeroSeq(t *testing.T) {
	reducer := func(a, b int) int { t.Error("reducer called"); return a }
	for _, p := range foldParallelism {
		got := foldOf(NewSeqFromSlice([]int{}, p), func() int { return 42 }, reducer, func(a, b int) int { return a })
		assert.Equal(t, 42, got, "p=%d", p)
	}
	assert.Equal(t, 42, foldOf(Seq[int]{}, func() int { return 42 }, reducer, func(a, b int) int { return a }))
}

// Every element is reduced exactly once.
func TestFold_ReducerCalledOncePerElement(t *testing.T) {
	for _, p := range foldParallelism {
		var calls atomic.Int64
		foldOf(NewSeqFromSlice(makeSlice(5000), p),
			func() int { return 0 },
			func(a, b int) int { calls.Add(1); return a + b },
			func(a, b int) int { return a + b })
		assert.EqualValues(t, 5000, calls.Load(), "p=%d", p)
	}
}

// With the identity as zero, merge is only needed for non-trivial combination: order-sensitive merge is not required,
// but the result of a commutative merge must not depend on which worker saw which element.
func TestFold_ResultIndependentOfWorkerSplit(t *testing.T) {
	for range 50 {
		for _, p := range []int{2, 8} {
			assert.Equal(t, 499500, foldSum(NewSeqFromSlice(makeSlice(1000), p)))
		}
	}
}

// Works in a pipeline after a parallel Map.
func TestFold_AfterMap(t *testing.T) {
	for _, p := range foldParallelism {
		got := foldSum(NewSeqFromSlice(makeSlice(100), p).Map(func(v int) int { return v * 2 }))
		assert.Equal(t, 9900, got, "p=%d", p)
	}
}

// merge must run on every worker's accumulator, so a zero()-sized contribution must not be double counted:
// zero() = 100 with sum would count 100 once per worker.
func TestFold_MergeCalledPerWorkerOnly(t *testing.T) {
	var merges atomic.Int64
	foldOf(NewSeqFromSlice(makeSlice(100), 4),
		func() int { return 0 },
		func(a, b int) int { return a + b },
		func(a, b int) int { merges.Add(1); return a + b })
	assert.EqualValues(t, 3, merges.Load(), "4 partials need 3 merges")
}

// A panic in a worker cannot be recovered by the caller; Fold must re-panic on the caller's goroutine.
func TestFold_ReducerPanicReachesCaller(t *testing.T) {
	boom := errors.New("boom")
	for _, p := range []int{1, 4} {
		var recovered any
		runWithTimeout(t, func() {
			defer func() { recovered = recover() }()
			foldOf(NewSeqFromSlice(makeSlice(100), p),
				func() int { return 0 },
				func(a, b int) int { panic(boom) },
				func(a, b int) int { return a + b })
		})
		assert.Equal(t, boom, recovered, "p=%d", p)
	}
}

// Parallelism below 1 must not crash.
func TestFold_ParallelismClamped(t *testing.T) {
	assert.Equal(t, 4950, foldSum(NewParallelSeq(NewSliceProducer(makeSlice(100)), 0)))
	assert.Equal(t, 4950, foldSum(NewParallelSeq(NewSliceProducer(makeSlice(100)), -3)))
}

func BenchmarkFold_VsAccumulate_Cheap(b *testing.B) {
	data := makeSlice(100_000)
	for _, p := range []int{1, 4, 8} {
		b.Run("fold/p="+string(rune('0'+p)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchSink = foldSum(NewSeqFromSlice(data, p))
			}
		})
		b.Run("accumulate/p="+string(rune('0'+p)), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				benchSink = NewSeqFromSlice(data, p).Accumulate(0, func(a, v int) int { return a + v })
			}
		})
	}
}

// foldOf adapts the method to a function form to keep call sites short.
func foldOf[T, R any](s Seq[T], zero func() R, reducer func(R, T) R, merge func(R, R) R) R {
	return s.Fold(zero, reducer, merge)
}
