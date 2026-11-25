// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"fmt"
	"testing"
)

// implementations under comparison
var reducers = []struct {
	name string
	run  func(s Seq[int], reducer func(int, int) int) int
}{
	{"mutex", func(s Seq[int], r func(int, int) int) int { return s.Reduce(r) }},
}

var reduceSink int

func benchReducer(b *testing.B, size int, parallelisms []int, build func(data []int, p int) Seq[int], reducer func(int, int) int) {
	data := makeSlice(size)
	for _, p := range parallelisms {
		for _, impl := range reducers {
			b.Run(fmt.Sprintf("p=%d/%s", p, impl.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					reduceSink = impl.run(build(data, p), reducer)
				}
			})
		}
	}
}

func benchReduce(b *testing.B, size int, parallelisms []int, build func(data []int, p int) Seq[int], reducer func(int, int) int) {
	data := makeSlice(size)
	for _, p := range parallelisms {
		for _, impl := range accumulators {
			b.Run(fmt.Sprintf("p=%d/%s", p, impl.name), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					benchSink = impl.run(build(data, p), reducer)
				}
			})
		}
	}
}

// Trivial reducer: measures pure synchronisation overhead (lock vs channel hand-off).
func BenchmarkReduce_CheapReducer(b *testing.B) {
	benchReducer(b, 100_000, []int{1, 2, 4, 8, 16}, plainSource, func(a, v int) int { return a + v })
}

// Reducer does real work. It is serialised in both implementations, so this shows how each copes with a slow consumer.
func BenchmarkReduce_ExpensiveReducer(b *testing.B) {
	benchReducer(b, 5_000, []int{1, 4, 8}, plainSource, func(a, v int) int { return a + spin(500) + v })
}

// Realistic pipeline: the expensive part is upstream (Map), the reducer is cheap. This is where parallelism should pay.
func BenchmarkReduce_ExpensiveUpstream(b *testing.B) {
	benchReducer(b, 5_000, []int{1, 4, 8}, func(data []int, p int) Seq[int] {
		return NewSeqFromSlice(data, p).Map(func(v int) int { return spin(50000) + v })
	}, func(a, v int) int { return a + v })
}

// Small input: goroutine and channel setup cost dominates.
func BenchmarkReduce_SmallInput(b *testing.B) {
	benchReducer(b, 16, []int{1, 4}, plainSource, func(a, v int) int { return a + v })
}
