// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

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
