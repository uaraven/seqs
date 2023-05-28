package seqs

import (
	"runtime"
	"sort"
	"sync"
)

type Seq[T any] interface {
	Parallelism() int
	Filter(test Predicate[T]) Seq[T]
	ToSlice() []T
	ToSortedSlice(comparator Comparator[T]) []T
	Next() (T, bool)
	First() Option[T]
	Rest() Seq[T]
	Reduce(func(a T, b T) T) T
}

type ProducerSeq[T any] struct {
	producer    Producer[T]
	parallelism int
}

func NewSeq[T any](producer Producer[T]) *ProducerSeq[T] {
	return &ProducerSeq[T]{producer: producer, parallelism: 1}
}

// NewParallelSeq creates a new Seq with a provided producer and the parallelism level
// If parallelism level is not specified, the number of CPUs will be used.
//
// All operations on the Seq will be performed in parallel. Ordering of elements in the
// resulting Seq is not guaranteed
//
// If the specified parallelism is equal to or less than 1, this will create a sequential Seq
func NewParallelSeq[T any](producer Producer[T], parallelism ...int) *ProducerSeq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{runtime.NumCPU()}
	}
	return &ProducerSeq[T]{producer: producer, parallelism: parallelism[0]}
}

func (t ProducerSeq[T]) Parallelism() int {
	return t.parallelism
}

func (t ProducerSeq[T]) Next() (T, bool) {
	v := t.producer()
	return v.Value(), v.IsPresent()
}

func (t ProducerSeq[T]) ToSlice() []T {
	result := make([]T, 0)
	t.ForEach(func(value T) {
		result = append(result, value)
	})
	return result
}

// ToSortedSlice converts the Seq to a slice where all elements are ordered according to
// the provided comparator function
//
// This is a terminal operation.
func (t ProducerSeq[T]) ToSortedSlice(comparator Comparator[T]) []T {
	data := t.ToSlice()
	sort.Slice(data, func(i, j int) bool {
		return comparator(data[i], data[j]) < 0
	})
	return data
}

func (t ProducerSeq[T]) Filter(test Predicate[T]) Seq[T] {
	return &ProducerSeq[T]{
		parallelism: t.parallelism,
		producer: func() Option[T] {
			for {
				v := t.producer()
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

func (t ProducerSeq[T]) Peek(consumer Consumer[T]) Seq[T] {
	return &ProducerSeq[T]{
		parallelism: t.parallelism,
		producer: func() Option[T] {
			for {
				v := t.producer()
				if !v.IsPresent() {
					return v
				}
				consumer(v.Value())
				return v
			}
		},
	}
}

func (t ProducerSeq[T]) ForEach(consumer Consumer[T]) {
	output := make(chan T)

	var wg sync.WaitGroup
	for i := 0; i < t.parallelism; i++ {
		wg.Add(1)
		go func() {
			v := t.producer()
			for v.IsPresent() {
				output <- v.Value()
				v = t.producer()
			}
			wg.Done()
		}()
	}

	// collector
	var cwg sync.WaitGroup
	cwg.Add(1)
	go func() {
		for v := range output {
			consumer(v)
		}
		cwg.Done()
	}()

	wg.Wait()
	close(output)
	cwg.Wait()
}

func (t ProducerSeq[T]) First() Option[T] {
	v, ok := t.Next()
	if ok {
		return SomeOf(v)
	} else {
		return NoneOf[T]()
	}
}

func (t ProducerSeq[T]) Rest() Seq[T] {
	t.Next() // skip the first element
	return t
}

func (t ProducerSeq[T]) Reduce(reducer func(a T, b T) T) T {
	output := make(chan T)

	var wg sync.WaitGroup
	for i := 0; i < t.parallelism; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			var v T
			vopt := t.producer()
			if vopt.IsPresent() {
				v = vopt.Value()
			} else {
				return
			}
			u := t.producer()
			for u.IsPresent() {
				v = reducer(v, u.Value())
				u = t.producer()
			}
			output <- v
		}()
	}

	// collector
	var cwg sync.WaitGroup
	cwg.Add(1)
	var result T
	go func() {
		v := <-output
		for u := range output {
			v = reducer(v, u)
		}
		result = v
		cwg.Done()
	}()

	wg.Wait()
	close(output)
	cwg.Wait()
	return result
}

// Map maps Seq elements using a mapper function. Map returns a new Seq and performs the transformation
// lazily.
func Map[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	if input.Parallelism() <= 1 {
		return sequentialMap(input, mapper)
	} else {
		return parallelMap(input, mapper)
	}
}

// FlatMap
func FlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	if input.Parallelism() <= 1 {
		return sequentialFlatMap(input, mapper)
	} else {
		return parallelFlatMap(input, mapper)
	}
}
