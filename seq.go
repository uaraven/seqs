// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
)

// Seq is the sequence of elements supporting sequential and parallel aggregate operations.
// Operations on sequences are composed into a sequence pipeline. The pipeline consists of a source (which can be a slice,
// channel or any Producer), zero or more intermediate operations, which transform a sequence into another sequence
// and a terminal operation, which produces a result or a side effect (such as Count() or ToSlice())
//
// Sequences are lazy. Computation on source data is performed only when the terminal operation is initiated and source
// elements are consumed as needed
//
// Functions performing intermediate operations must be side effect-free and must not modify the sequence source
type Seq[T any] struct {
	producer    Producer[T]
	parallelism int
}

// isZero returns true for the zero value of Seq, which is treated as an empty sequence
func (t Seq[T]) isZero() bool {
	return t.producer == nil
}

// NewParallelSeq creates a new Seq that will execute operations in parallel manner.
// If parallelism level is not specified, then the parallelism will be set to the number of CPUs.
//
// All operations on the Seq will be performed in parallel. Ordering of elements in the
// resulting Seq is not guaranteed
//
// If the specified parallelism is equal to or less than 1, this will create a sequential Seq
func NewParallelSeq[T any](producer Producer[T], parallelism ...int) Seq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{runtime.NumCPU()}
	}
	if parallelism[0] <= 0 {
		parallelism[0] = 1
	}
	return Seq[T]{producer: producer, parallelism: parallelism[0]}
}

// Parallelism returns the number of goroutines used to perform operations on this Seq pipeline
func (t Seq[T]) Parallelism() int {
	if t.parallelism < 1 {
		return 1
	}
	return t.parallelism
}

// Next returns the next element in this Seq or an empty Option if the Seq is exhausted.
func (t Seq[T]) Next() Option[T] {
	if t.isZero() {
		return NoneOf[T]()
	}
	return t.producer()
}

func (t Seq[T]) ToSlice() []T {
	result := make([]T, 0)
	t.ForEach(func(value T) {
		result = append(result, value)
	})
	return result
}

func (t Seq[T]) ToSortedSlice(comparator Comparator[T]) []T {
	data := t.ToSlice()
	sort.Slice(data, func(i, j int) bool {
		return comparator(data[i], data[j]) < 0
	})
	return data
}

func (t Seq[T]) Filter(test Predicate[T]) Seq[T] {
	return Seq[T]{
		parallelism: t.parallelism,
		producer: func() Option[T] {
			for {
				v := t.Next()
				if !v.IsPresent() {
					return v
				}
				if test(v.Value()) {
					return v
				}
			}
		},
	}
}

func (t Seq[T]) Peek(consumer Consumer[T]) Seq[T] {
	return Seq[T]{
		parallelism: t.parallelism,
		producer: func() Option[T] {
			for {
				v := t.Next()
				v.IfPresent(consumer)
				return v
			}
		},
	}
}

func (t Seq[T]) ForEach(consumer Consumer[T]) {
	output := make(chan T)

	var wg sync.WaitGroup
	for i := 0; i < t.Parallelism(); i++ {
		wg.Go(func() {
			v := t.Next()
			for v.IsPresent() {
				output <- v.Value()
				v = t.Next()
			}
		})
	}

	// collector
	var cwg sync.WaitGroup
	cwg.Add(1)
	go func() {
		defer cwg.Done()
		for v := range output {
			consumer(v)
		}
	}()

	wg.Wait()
	close(output)
	cwg.Wait()
}

func (t Seq[T]) First() Option[T] {
	return t.Next()
}

func (t Seq[T]) Rest() Seq[T] {
	return t.Skip(1)
}

func (t Seq[T]) Take(count int64) Seq[T] {
	taken := int64(0)
	return NewParallelSeq(
		func() Option[T] {
			if atomic.AddInt64(&taken, 1) > count {
				return NoneOf[T]()
			}
			return t.Next()
		},
		t.Parallelism(),
	)
}

func (t Seq[T]) Skip(count int64) Seq[T] {
	var mtx sync.Mutex
	var skipped atomic.Bool
	return NewParallelSeq(
		func() Option[T] {
			if !skipped.Load() {
				mtx.Lock()
				if skipped.Load() {
					mtx.Unlock()
					return t.Next()
				}
				for range count {
					v := t.Next()
					if !v.IsPresent() {
						skipped.Store(true)
						mtx.Unlock()
						return v
					}
				}
				skipped.Store(true)
				mtx.Unlock()
			}
			return t.Next()
		},
		t.Parallelism(),
	)
}

func (t Seq[T]) Reduce(reducer func(a T, b T) T) T {
	output := make(chan T)

	var wg sync.WaitGroup
	for i := 0; i < t.Parallelism(); i++ {
		wg.Go(func() {
			var v T
			vopt := t.Next()
			if vopt.IsPresent() {
				v = vopt.Value()
			} else {
				return
			}
			u := t.Next()
			for u.IsPresent() {
				v = reducer(v, u.Value())
				u = t.Next()
			}
			output <- v
		})
	}

	// collector
	var cwg sync.WaitGroup
	var result T
	cwg.Go(func() {
		v := <-output
		for u := range output {
			v = reducer(v, u)
		}
		result = v
	})

	wg.Wait()
	close(output)
	cwg.Wait()
	return result
}

func (t Seq[T]) Count() int64 {
	return t.Accumulate(int64(0), func(a int64, b T) int64 {
		return a + 1
	})
}

func (t Seq[T]) AnyMatch(test Predicate[T]) bool {
	found := atomic.Bool{}

	var wg sync.WaitGroup
	for i := 0; i < t.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v := t.Next()
			for v.IsPresent() {
				if found.Load() {
					break
				}
				if test(v.Value()) {
					found.Store(true)
					return
				}
				v = t.Next()
			}
		}()
	}

	wg.Wait()
	return found.Load()
}

func (t Seq[T]) AllMatch(test Predicate[T]) bool {
	return !t.AnyMatch(func(e T) bool {
		return !test(e)
	})
}

func (t Seq[T]) Ordered(comparator Comparator[T]) Seq[T] {
	return NewSeqFromSlice(t.ToSortedSlice(comparator), 1)
}

func (t Seq[T]) Parallel(parallelism ...int) Seq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{runtime.NumCPU()}
	}
	if parallelism[0] == t.Parallelism() {
		return t
	}
	return NewParallelSeq(t.producer, parallelism...)
}

func (t Seq[T]) Sequential() Seq[T] {
	if t.Parallelism() == 1 {
		return t
	}
	return NewParallelSeq(t.producer, 1)
}

// TODO:
// Partition Seq[T] -> Seq[Seq[T]]
// SinkToChannel(chan)
