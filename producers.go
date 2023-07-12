package seqs

import (
	"sync/atomic"
)

// Producer is a function that produces a value each time it is called, creating a potentially endless stream of
// values.
//
// Producer returns a Some if there is a new value produced or a None if there is no more values to be produced.
// After producer has returned empty Option, each subsequent invocation must return empty Option as well
//
// Producer is used as a source for Seq
type Producer[T any] func() Option[T]

// Consumer represents an operation that accepts a single input argument and returns no result.
type Consumer[T any] func(value T)

// Provider represents a function that provides a result.
// There is no requirement that a new or distinct result be returned each time the provider is invoked.
type Provider[T any] func() T

// Comparator represents a function that compares two values t1 and t2
// Comparator returns
//
// -1 if t1 < t2,
// 1 if t1 > t2, and
// 0, if t1 == t2
type Comparator[T any] func(t1 T, t2 T) int

// Predicate represents a boolean-valued function of one argument.
type Predicate[T any] func(value T) bool

// NewSliceProducer creates a Producer that returns each value of the slice with each new invocation
func NewSliceProducer[T any](data []T) Producer[T] {
	var index atomic.Int32
	index.Store(-1)
	return func() Option[T] {
		idx := index.Add(1)
		if idx < int32(len(data)) {
			return SomeOf(data[idx])
		} else {
			return NoneOf[T]()
		}
	}
}

// NewChannelProducer creates a Producer that returns value received from the channel with each new invocation
// This producer will block if channel is not ready
func NewChannelProducer[T any](source chan T) Producer[T] {
	return func() Option[T] {
		select {
		case data, ok := <-source:
			if !ok {
				return NoneOf[T]()
			} else {
				return SomeOf(data)
			}
		}
	}
}
