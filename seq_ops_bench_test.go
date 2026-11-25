// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"fmt"
	"testing"
)

// implementations under comparison
var accumulators = []struct {
	name string
	run  func(s Seq[int], reducer func(int, int) int) int
}{
	{"mutex", func(s Seq[int], r func(int, int) int) int { return s.Accumulate(0, r) }},
}

var benchSink int

// spin burns a roughly fixed amount of CPU without being optimised away.
func spin(n int) int {
	x := 1
	for i := 0; i < n; i++ {
		x = x*31 + i
	}
	return x
}

func benchAccumulators(b *testing.B, size int, parallelisms []int, build func(data []int, p int) Seq[int], reducer func(int, int) int) {
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

func plainSource(data []int, p int) Seq[int] { return NewSeqFromSlice(data, p) }

// Trivial reducer: measures pure synchronisation overhead (lock vs channel hand-off).
func BenchmarkAccumulate_CheapReducer(b *testing.B) {
	benchAccumulators(b, 100_000, []int{1, 2, 4, 8, 16}, plainSource, func(a, v int) int { return a + v })
}

// Reducer does real work. It is serialised in both implementations, so this shows how each copes with a slow consumer.
func BenchmarkAccumulate_ExpensiveReducer(b *testing.B) {
	benchAccumulators(b, 5_000, []int{1, 4, 8}, plainSource, func(a, v int) int { return a + spin(500) + v })
}

// Realistic pipeline: the expensive part is upstream (Map), the reducer is cheap. This is where parallelism should pay.
func BenchmarkAccumulate_ExpensiveUpstream(b *testing.B) {
	benchAccumulators(b, 5_000, []int{1, 4, 8}, func(data []int, p int) Seq[int] {
		return NewSeqFromSlice(data, p).Map(func(v int) int { return spin(50000) + v })
	}, func(a, v int) int { return a + v })
}

// Small input: goroutine and channel setup cost dominates.
func BenchmarkAccumulate_SmallInput(b *testing.B) {
	benchAccumulators(b, 16, []int{1, 4}, plainSource, func(a, v int) int { return a + v })
}
