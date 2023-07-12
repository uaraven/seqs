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
type Seq[T any] interface {
	// Parallelism returns the number of goroutines used to perform operations on this Seq pipeline
	Parallelism() int
	// Filter returns a new Seq containing elements of this Seq that match the give predicate
	Filter(test Predicate[T]) Seq[T]
	// ToSlice returns a slice of all the elements of this Seq. This is a terminal operation that will execute the
	// prepared Seq pipeline
	ToSlice() []T
	// ToSortedSlice returns a slice of all the elements of this Seq. The elements in the resulting slice will be
	// ordered according to the provided comparison function.
	// This is a terminal operation that will execute the prepared Seq pipeline
	ToSortedSlice(comparator Comparator[T]) []T
	// Next returns the next element in this Seq or an zero value of the type T. Second return value is true if
	// the last call to Next exhausted this Seq.
	Next() (T, bool)
	// First returns the fist element of this Seq or None if the Seq is empty
	First() Option[T]
	// Rest returns a new Seq containing all the elements of this Seq except for the first one
	Rest() Seq[T]
	// Reduce applies provided function to the first two elements of this Seq and then iteratively to the result of
	// the previous reduction and the next element of this Seq until the Seq is exhausted. It returns the result of the
	// reduction.
	// Reduce is a terminal operation
	Reduce(func(a T, b T) T) T
	// ForEach will apply the provided consumer to each element of this Seq. The order of iteration is not guaranteed
	// This is a terminal operation
	ForEach(consumer Consumer[T])
	// Take returns a new Seq consisting of the first N elements of this Seq
	Take(count int64) Seq[T]
	// Skip returns a new Seq consisting of all the elements of this Seq except for the first N elements
	Skip(count int64) Seq[T]
	// Count returns the number of elements in this Seq. This is a terminal operation
	Count() int64
	// Ordered returns a new Seq containing all the elements of this Seq in order.
	// This operation will trigger the execution of the Seq pipeline. It requires O(N) memory and returned Seq will
	// be a sequential, even if this Seq is parallel
	Ordered(comparator Comparator[T]) Seq[T]
	// Parallel creates a new Seq with parallel pipeline and number of goroutines equal to the provided parallelism value
	// If parallelism is not provided it will be set to the number of CPU cores
	// If the new parallelism number is equal to parallelism of this Seq, then this Seq is returned unchanged
	Parallel(parallelism ...int) Seq[T]
	// Sequential creates a new Seq with sequential pipeline. If this Seq is a parallel Seq, then the order of elements
	// in the returned Seq is not guaranteed.
	// If this Seq is sequential, then it is returned unchanged
	Sequential() Seq[T]
}

type ProducerSeq[T any] struct {
	producer    Producer[T]
	parallelism int
}

// EmptySeq creates a new sequence that has no elements
func EmptySeq[T any]() *ProducerSeq[T] {
	return &ProducerSeq[T]{
		producer: func() Option[T] {
			return NoneOf[T]()
		},
		parallelism: 1,
	}
}

// SeqOf returns a sequential Seq of provided values
func SeqOf[T any](values ...T) *ProducerSeq[T] {
	return NewSeq(NewSliceProducer(values))
}

// NewSeq creates a new sequence that executes operations sequentially
func NewSeq[T any](producer Producer[T]) *ProducerSeq[T] {
	return &ProducerSeq[T]{producer: producer, parallelism: 1}
}

// NewSeqFromSlice creates a new sequence from the given slice
//
// If optional parallelism value is provided, and it is higher than 1, then the created Seq will perform
// operations in parallel.
func NewSeqFromSlice[T any](source []T, parallelism ...int) *ProducerSeq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{1}
	}
	return NewParallelSeq(NewSliceProducer(source), parallelism...)
}

func NewSeqFromMapKeys[K comparable, V any](source map[K]V, parallelism ...int) *ProducerSeq[K] {
	if len(parallelism) == 0 {
		parallelism = []int{1}
	}
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
	if len(parallelism) == 0 {
		parallelism = []int{1}
	}
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
				v.IfPresent(consumer)
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
	return t.Skip(1)
}

func (t ProducerSeq[T]) Take(count int64) Seq[T] {
	counter := int64(0)
	return NewParallelSeq(
		func() Option[T] {
			if counter < count {
				atomic.AddInt64(&counter, 1)
				return t.producer()
			} else {
				return NoneOf[T]()
			}
		},
		t.Parallelism(),
	)
}

func (t ProducerSeq[T]) Skip(count int64) Seq[T] {
	skipped := atomic.Bool{}
	return NewParallelSeq(
		func() Option[T] {
			// if already skipped, return next value immediately
			if skipped.Load() {
				return t.producer()
			}
			// set skipped flag
			skipped.Store(true)
			// try to skip requested number of elements
			for i := int64(0); i < count; i++ {
				v := t.producer()
				if !v.IsPresent() { // if reached the end of Seq, return immediately
					return v
				}
			}
			return t.producer() // return next value
		},
		t.Parallelism(),
	)
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

func (t ProducerSeq[T]) Count() int64 {
	return Accumulate[T, int64](t, int64(0), func(a int64, b T) int64 {
		return a + 1
	})
}

func (t ProducerSeq[T]) Ordered(comparator Comparator[T]) Seq[T] {
	return NewSeqFromSlice(t.ToSortedSlice(comparator), 1)
}

func (t ProducerSeq[T]) Parallel(parallelism ...int) Seq[T] {
	if len(parallelism) == 0 {
		parallelism = []int{runtime.NumCPU()}
	}
	if parallelism[0] == t.parallelism {
		return t
	}
	return NewParallelSeq(t.producer, parallelism...)
}

func (t ProducerSeq[T]) Sequential() Seq[T] {
	if t.parallelism == 1 {
		return t
	}
	return NewParallelSeq(t.producer, 1)
}

// Accumulate reduces a sequence to a single value, combining each element of the sequence with a previous value
// of the accumulator
//
// Initial value of the accumulator is passed as an argument
func Accumulate[T any, R any](input Seq[T], initial R, reducer func(a R, b T) R) R {
	output := make(chan T)

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, ok := input.Next()
			for ok {
				output <- v
				v, ok = input.Next()
			}
		}()
	}

	// collector
	var cwg sync.WaitGroup
	cwg.Add(1)
	accumulator := initial
	go func() {
		for u := range output {
			accumulator = reducer(accumulator, u)
		}
		cwg.Done()
	}()

	wg.Wait()
	close(output)
	cwg.Wait()
	return accumulator
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
// If a mapped sequence is nil an empty sequence is used, instead.
//
// FlatMap returns a new Seq and performs the transformation lazily.
func FlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	if input.Parallelism() <= 1 {
		return sequentialFlatMap(input, mapper)
	} else {
		return parallelFlatMap(input, mapper)
	}
}

// TODO:
// GroupBy
// TakeWhile
// SkipWhile
// Partition Seq[T] -> Seq[Seq[T]]
// SinkToChannel(chan)
// FindFirst
// AllMatch
// AnyMatch
