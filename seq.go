package seqs

import (
	"runtime"
	"sort"
	"sync"
)

// Seq is the sequence of elements supporting sequential and parallel aggregate operations
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

// NewSeq creates a new sequence that executes operations sequentially
func NewSeq[T any](producer Producer[T]) *ProducerSeq[T] {
	return &ProducerSeq[T]{producer: producer, parallelism: 1}
}

// NewSeqFromSlice creates a new sequence from the given slice
//
// If optional parallelism value is provided and it is higher than 1, then the created Seq will perform operations in parallel.
func NewSeqFromSlice[T any](source []T, parallelism ...int) *ProducerSeq[T] {
	return NewParallelSeq(NewSliceProducer(source), parallelism...)
}

func NewSeqFromMapKeys[K comparable, V any](source map[K]V, parallelism ...int) *ProducerSeq[K] {
	keyChan := make(chan K)
	go func() {
		for k := range source {
			keyChan <- k
		}
		close(keyChan)
	}()
	return NewParallelSeq(NewChannelProducer(keyChan), parallelism...)
}

func NewSeqFromMapValues[K comparable, V any](source map[K]V, parallelism ...int) *ProducerSeq[V] {
	valChan := make(chan V)
	go func() {
		for _, v := range source {
			valChan <- v
		}
		close(valChan)
	}()
	return NewParallelSeq(NewChannelProducer(valChan), parallelism...)
}

// NewParallelSeq creates a new Seq that will execute operations in parallel manner.
// If parallelism level is not specified, then the parallelism will be set to the number of CPUs.
//
// All operations on the Seq will be performed in parallel. Ordering of elements in the
// resulting Seq is not guaranteed
//
// If the specified parallelism is equal to or less than 1, this will create a sequential Seq
func NewParallelSeq[T any](producer Producer[T], parallelism ...int) *ProducerSeq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{runtime.NumCPU()}
	}
	if parallelism[0] <= 0 {
		parallelism[0] = 1
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

// Map returns a Seq consisting of the results of applying the given mapping function to the elements for the provided input sequence
//
// Map returns a new Seq and performs the transformation lazily.
func Map[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	if input.Parallelism() <= 1 {
		return sequentialMap(input, mapper)
	} else {
		return parallelMap(input, mapper)
	}
}

// FlatMap returns a sequence consisting of the results of replacing each element of the provided input sequence with
// the contents of a mapped sequence produced by applying the provided mapping function to each element.
//
// If a mapped sequence is nil an empty stream is used, instead.
//
// FlatMap returns a new Seq and performs the transformation lazily.
func FlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	if input.Parallelism() <= 1 {
		return sequentialFlatMap(input, mapper)
	} else {
		return parallelFlatMap(input, mapper)
	}
}
