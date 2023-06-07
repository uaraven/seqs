package seqs

import (
	"sync/atomic"
)

type Producer[T any] func() Option[T]

type Consumer[T any] func(value T)

type Provider[T any] func() T

type Comparator[T any] func(t1 T, t2 T) int

type Predicate[T any] func(value T) bool

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
