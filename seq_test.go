// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"github.com/stretchr/testify/assert"
	"runtime"
	"slices"
	"testing"
	"time"
)

func TestSliceFilter(t *testing.T) {
	slice := makeSlice(1000)

	seq := NewSeq(NewSliceProducer(slice))
	nseq := seq.Filter(func(value int) bool {
		return value%2 == 0
	})
	result := nseq.ToSlice()

	slices.Sort(result)

	expected := make([]int, 0)
	for i := range len(slice) {
		if i%2 == 0 {
			expected = append(expected, i)
		}
	}

	assert.EqualValues(t, expected, result)
}

func TestSliceParallelFilter(t *testing.T) {
	slice := makeSlice(1000)

	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	nseq := seq.Filter(func(value int) bool {
		return value%2 == 0
	})
	result := nseq.ToSlice()

	slices.Sort(result)

	expected := make([]int, 0)
	for i := range len(slice) {
		if i%2 == 0 {
			expected = append(expected, i)
		}
	}

	assert.EqualValues(t, expected, result)
}

func TestMap(t *testing.T) {
	data := makeSlice(1000)
	seq := NewParallelSeq(NewSliceProducer(data), runtime.NumCPU())
	mapped := seq.Map(func(t int) int {
		return t * 2
	}).ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})

	expected := make([]int, len(data))
	for i := range len(data) {
		expected[i] = data[i] * 2
	}

	assert.EqualValues(t, expected, mapped)
}

func TestFlatMap(t *testing.T) {
	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{9},
	}
	seq := NewSeq(NewSliceProducer(superSlice))
	flatMap := seq.FlatMap(func(t []int) Seq[int] {
		return NewSeq(NewSliceProducer(t))
	})
	result := flatMap.ToSlice()
	assert.EqualValues(t, []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}, result)
}

func TestFlatMapWithNilMap(t *testing.T) {
	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{},
		{9},
	}
	seq := NewSeq(NewSliceProducer(superSlice))
	flatMap := seq.FlatMap(func(t []int) Seq[int] {
		if len(t) == 0 {
			return Seq[int]{}
		}
		return NewSeq(NewSliceProducer(t))
	})
	result := flatMap.ToSlice()
	assert.EqualValues(t, []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}, result)
}

func TestParallelFlatMap(t *testing.T) {
	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{9},
	}
	seq := NewParallelSeq(NewSliceProducer(superSlice), 4)
	flatMap := seq.FlatMap(func(t []int) Seq[int] {
		return NewSeq(NewSliceProducer(t))
	})
	result := flatMap.ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})
	assert.EqualValues(t, []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}, result)
}

func TestParallelFlatMapWithNilMap(t *testing.T) {
	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{},
		{9},
	}
	seq := NewParallelSeq(NewSliceProducer(superSlice), 4)
	flatMap := seq.FlatMap(func(t []int) Seq[int] {
		if len(t) == 0 {
			return Seq[int]{}
		}
		return NewSeq(NewSliceProducer(t))
	})
	result := flatMap.ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})
	assert.EqualValues(t, []int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}, result)
}

func TestProducerSeq_First(t *testing.T) {
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.First()
	assert.True(t, v.IsPresent())
	assert.Equal(t, 0, v.Value())
	slice = makeSlice(0)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.First()
	assert.False(t, v.IsPresent())
}

func TestProducerSeq_Rest(t *testing.T) {
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Rest().ToSlice()
	containsAll(t, v, 1, 2, 3, 4)
	slice = makeSlice(0)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.Rest().ToSlice()
	assert.Len(t, v, 0)
}

func TestProducerSeq_Reduce(t *testing.T) {
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Reduce(func(a int, b int) int {
		return a + b
	})

	assert.Equal(t, 10, v)

	slice = makeSlice(51)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.Reduce(func(a int, b int) int {
		return a + b
	})

	assert.Equal(t, 1275, v)
}

func TestProducerSeq_Accumulate(t *testing.T) {
	slice := []string{"a", "bb", "ccc", "dddd", "eeeee"}
	seq := NewSeqFromSlice(slice, 4)
	v := seq.Accumulate(0, func(a int, b string) int {
		return a + len(b)
	})

	assert.Equal(t, 15, v)
}

func TestProducerSeq_ReduceWithChannelProducer(t *testing.T) {
	slice := makeSlice(51)
	src := make(chan int)
	seq := NewParallelSeq(NewChannelProducer(src), 4)

	go func() {
		for _, i := range slice {
			src <- i
		}
		close(src)
	}()

	v := seq.Reduce(func(a int, b int) int {
		return a + b
	})

	assert.Equal(t, 1275, v)
}

func TestNewSeqFromMapKeys(t *testing.T) {
	src := map[int]string{
		4: "four",
		2: "two",
		1: "one",
		3: "three",
	}
	keys := NewSeqFromMapKeys(src).ToSlice()
	assert.Len(t, keys, 4)
	containsAll(t, keys, 1, 2, 3, 4)
}

func TestNewSeqFromMapValues(t *testing.T) {
	src := map[int]string{
		4: "four",
		2: "two",
		1: "one",
		3: "three",
	}
	vals := NewSeqFromMapValues(src).ToSlice()
	assert.Len(t, vals, 4)
	containsAll(t, vals, "one", "two", "three", "four")
}

func TestProducerSeq_Count(t *testing.T) {
	slice := makeSlice(51)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Count()

	assert.Equal(t, int64(51), v)
}

func TestEmptySeq(t *testing.T) {
	r := EmptySeq[int]().ToSlice()

	assert.Len(t, r, 0)
}

func TestProducerSeq_Peek(t *testing.T) {
	src := SeqOf(1, 2, 3, 4, 5)
	dst := make([]int, 0)

	src.Peek(func(value int) {
		dst = append(dst, value)
	}).Count()

	assert.Len(t, dst, 5)
	containsAll(t, dst, 1, 2, 3, 4, 5)
}

func TestProducerSeq_Take(t *testing.T) {
	res1 := SeqOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Take(4).ToSlice()
	containsAll(t, res1, 1, 2, 3, 4)

	res2 := SeqOf(1, 2, 3, 4).Take(6).ToSlice()
	containsAll(t, res2, 1, 2, 3, 4)
}

func TestProducerSeq_Skip(t *testing.T) {
	res1 := SeqOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Skip(4).ToSlice()
	containsAll(t, res1, 5, 6, 7, 8, 9, 10)

	res2 := SeqOf(1, 2, 3, 4).Skip(6).ToSlice()
	assert.Len(t, res2, 0)

	res3 := SeqOf(1, 2, 3, 4).Skip(4).ToSlice()
	assert.Len(t, res3, 0)
}

// Regression test: in parallel mode Skip used to flag skipping as done before it finished, so workers returned
// elements that should have been skipped, and the result length was wrong.
func TestProducerSeq_SkipParallelCount(t *testing.T) {
	const size = 1000
	for _, parallelism := range []int{1, 2, 4, 8, 32} {
		for _, skip := range []int64{0, 1, 2, 5, 500, 999, 1000, 2000} {
			want := max(int64(size)-skip, 0)
			for range 10 {
				runWithTimeout(t, func() {
					res := NewSeqFromSlice(makeSlice(size), parallelism).Skip(skip).ToSlice()
					assert.EqualValues(t, want, len(res), "parallelism=%d skip=%d", parallelism, skip)
					// every source element is returned at most once
					seen := make(map[int]bool, len(res))
					for _, v := range res {
						assert.False(t, seen[v], "duplicate %d, parallelism=%d skip=%d", v, parallelism, skip)
						seen[v] = true
					}
				})
			}
		}
	}
}

func TestProducerSeq_SkipParallelCountOperation(t *testing.T) {
	for _, parallelism := range []int{1, 4, 8} {
		runWithTimeout(t, func() {
			assert.EqualValues(t, 500, NewSeqFromSlice(makeSlice(1000), parallelism).Skip(500).Count())
		})
	}
}

// A sequential Seq must skip exactly the first N elements, in order.
func TestProducerSeq_SkipSequentialOrder(t *testing.T) {
	assert.Equal(t, []int{1, 2, 3, 4}, SeqOf(0, 1, 2, 3, 4).Skip(1).ToSlice())
	assert.Equal(t, []int{5, 6, 7, 8, 9}, SeqOf(0, 1, 2, 3, 4, 5, 6, 7, 8, 9).Skip(5).ToSlice())
	assert.Equal(t, []int{0, 1, 2}, SeqOf(0, 1, 2).Skip(0).ToSlice())
	assert.Empty(t, SeqOf(0, 1, 2).Skip(3).ToSlice())
	assert.Empty(t, Seq[int]{}.Skip(2).ToSlice())
}

func TestProducerSeq_Ordered(t *testing.T) {
	src := NewSeqFromSlice([]int{10, 2, 5, 9, 3, 7, 6, 8, 4, 1}, 4)
	res := src.Ordered(func(t1 int, t2 int) int {
		return t1 - t2
	}).ToSlice()

	assert.Equal(t, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, res)
}

func TestProducerSeq_Parallel(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	src := NewSeqFromSlice(slice)

	assert.Equal(t, slice, src.ToSlice())

	psrc := src.Parallel(4)

	assert.NotEqual(t, slice, psrc.ToSlice())
	assert.Equal(t, 4, psrc.Parallelism())
}

func TestProducerSeq_Sequential(t *testing.T) {
	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	src := NewSeqFromSlice(slice, 4)

	ssrc := src.Sequential()
	assert.Subset(t, ssrc.ToSlice(), slice)
	assert.Equal(t, 1, ssrc.Parallelism())
}

func TestToMultiMap(t *testing.T) {
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 2)
	result := seq.ToMultiMap(func(e string) int { return len(e) }, func(e string) string { return e })

	assert.Len(t, result, 5)
	assert.Subset(t, result[1], []string{"a", "i"})
	assert.Subset(t, result[2], []string{"bb", "hh"})
	assert.Subset(t, result[3], []string{"ccc", "ggg"})
	assert.Subset(t, result[4], []string{"dddd", "ffff"})
	assert.Subset(t, result[5], []string{"eeeee"})
}

func TestToMap(t *testing.T) {
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)
	result := seq.ToMap(
		func(e string) int { return len(e) },
		func(e string) string { return e },
		func(k int, v1 string, v2 string) string {
			if v1 < v2 {
				return v1
			} else {
				return v2
			}
		})

	assert.EqualValues(t, map[int]string{
		1: "a",
		2: "bb",
		3: "ccc",
		4: "dddd",
		5: "eeeee",
	}, result)
}

func TestProducerSeq_AnyMatch(t *testing.T) {
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)

	result := seq.AnyMatch(func(value string) bool {
		return value == "ggg"
	})
	assert.True(t, result)

	seq = NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)
	result = seq.AnyMatch(func(value string) bool {
		return value == "xyz"
	})
	assert.False(t, result)
}

func TestProducerSeq_AllMatch(t *testing.T) {
	seq := NewSeqFromSlice(makeSlice(100), 4)

	result := seq.AllMatch(func(value int) bool {
		return value < 1000
	})
	assert.True(t, result)

	seq = NewSeqFromSlice(makeSlice(100), 4)
	result = seq.AllMatch(func(value int) bool {
		return value != 50
	})
	assert.False(t, result)
}

// runWithTimeout fails the test if f does not return in time. It guards against deadlocks.
func runWithTimeout(t *testing.T, f func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		f()
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("operation did not complete in time: deadlock")
	}
}

// Regression test: with parallelism > 1, several workers matching used to block forever on an unbuffered channel.
func TestProducerSeq_AnyMatchParallelNoDeadlock(t *testing.T) {
	data := makeSlice(100)
	for _, parallelism := range []int{1, 2, 4, 8, 32} {
		for range 50 {
			runWithTimeout(t, func() {
				// every element matches, so every worker finds a match
				assert.True(t, NewSeqFromSlice(data, parallelism).AnyMatch(func(int) bool { return true }))
				assert.False(t, NewSeqFromSlice(data, parallelism).AnyMatch(func(int) bool { return false }))
				assert.True(t, NewSeqFromSlice(data, parallelism).AnyMatch(func(v int) bool { return v == 99 }))
			})
		}
	}
}

// Regression test: AllMatch is built on AnyMatch and used to hang when two or more elements failed the test.
func TestProducerSeq_AllMatchParallelNoDeadlock(t *testing.T) {
	data := makeSlice(100)
	for _, parallelism := range []int{1, 2, 4, 8, 32} {
		for range 50 {
			runWithTimeout(t, func() {
				// many elements fail the test
				assert.False(t, NewSeqFromSlice(data, parallelism).AllMatch(func(v int) bool { return v%2 == 0 }))
				assert.False(t, NewSeqFromSlice(data, parallelism).AllMatch(func(int) bool { return false }))
				assert.True(t, NewSeqFromSlice(data, parallelism).AllMatch(func(v int) bool { return v >= 0 }))
			})
		}
	}
}

func TestProducerSeq_AnyAllMatchEmpty(t *testing.T) {
	for _, parallelism := range []int{1, 4} {
		runWithTimeout(t, func() {
			assert.False(t, NewSeqFromSlice([]int{}, parallelism).AnyMatch(func(int) bool { return true }))
			assert.True(t, NewSeqFromSlice([]int{}, parallelism).AllMatch(func(int) bool { return false }))
		})
	}
	assert.False(t, Seq[int]{}.AnyMatch(func(int) bool { return true }))
}

func makeSlice(n int) []int {
	slice := make([]int, n)
	for i := range len(slice) {
		slice[i] = i
	}
	return slice
}

// func TestMapPerf(t *testing.T) {
//	testSequential := func() {
//		micros := int64(0)
//
//		data := makeSlice(10000)
//		for i := range 10 {
//			start := time.Now()
//			seq := NewSeq(NewSliceProducer(data))
//			Map[int, int](seq, func(t int) int {
//				time.Sleep(time.Microsecond)
//				return t * 2
//			}).ToSlice()
//			elapsed := time.Now().Sub(start)
//			micros = micros + elapsed.Microseconds()
//		}
//		total := float64(micros) / 100.0
//		fmt.Printf("Sequential Map: %5.4f\n", total)
//	}
//
//	testParallel := func() {
//		micros := int64(0)
//
//		data := makeSlice(10000)
//		for i := range 10 {
//			start := time.Now()
//			seq := NewParallelSeq(NewSliceProducer(data), 4)
//			Map[int, int](seq, func(t int) int {
//				time.Sleep(time.Microsecond)
//				return t * 2
//			}).ToSlice()
//			elapsed := time.Now().Sub(start)
//			micros = micros + elapsed.Microseconds()
//		}
//		total := float64(micros) / 100.0
//		fmt.Printf("Parallel Map, 4 goroutines: %5.4f\n", total)
//	}
//
//	testSequential()
//	testParallel()
// }
