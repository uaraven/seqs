// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import "fmt"

// Option is a container that may or may not contain a value. If the value is present, then IsPresent() will return true
// and Value() will return the value
//
// Option can have two states: some and none. Some contains a value while none is an Option without a value.
// The zero value of Option is an empty Option.
//
// Create new Options using SomeOf() and NoneOf() functions
type Option[T any] struct {
	value   T
	present bool
}

// SomeOf creates a new Option[T] wrapping a provided value
func SomeOf[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// NoneOf creates a new empty Option[T]
func NoneOf[T any]() Option[T] {
	return Option[T]{}
}

// IsPresent returns true if Option contains a value
func (o Option[T]) IsPresent() bool {
	return o.present
}

// IfPresent accepts a Consumer function that will be executed if this Option has a value.
// The value will be passed to the consumer
//
// If the Option is empty, consumer will not be called
func (o Option[T]) IfPresent(consumer Consumer[T]) {
	if o.present {
		consumer(o.value)
	}
}

// Value returns the value held in the Option.
// Note: If Option is empty, the zero value of the type T will be returned
func (o Option[T]) Value() T {
	return o.value
}

// OrElse returns the value of the Option if it has one or the value provided as a parameter if the Option is empty
func (o Option[T]) OrElse(v T) T {
	if o.present {
		return o.value
	}
	return v
}

// OrElseGet is similar to OrElse, but it uses a Provider function to generate a returned value of Option is empty
func (o Option[T]) OrElseGet(provider Provider[T]) T {
	if o.present {
		return o.value
	}
	return provider()
}

// Apply accepts two functions as a parameters. onPresent is called if the Option contains a value, and
// onAbsent is called if the Option is empty
func (o Option[T]) Apply(onPresent Consumer[T], onAbsent func()) {
	if o.present {
		onPresent(o.value)
	} else {
		onAbsent()
	}
}

// ToSeq converts the Option to a Seq. Resulting Seq will either contain one value, in case of Option containing
// a value, or it will be an empty Seq, in case the Option is empty
func (o Option[T]) ToSeq() Seq[T] {
	if o.present {
		return SeqOf(o.value)
	}
	return EmptySeq[T]()
}

// Unwrap this Option into two values - the value contained in the Option (or zero value of type T) and the boolean
// indicating whether the Option was not empty
func (o Option[T]) Unwrap() (T, bool) {
	return o.value, o.present
}

func (o Option[T]) String() string {
	if o.present {
		return fmt.Sprintf("Some[%v]", o.value)
	}
	return "None[]"
}

// Map performs a map operation on the Option. If this Option is non-empty, then mapper function is called on the
// Option value and new Option of the different type is returned
//
// If this Option is empty then new empty Option[U] is returned
func (o Option[T]) Map[U any](mapper func(T) U) Option[U] {
	if o.present {
		return SomeOf(mapper(o.value))
	}
	return NoneOf[U]()
}

// FlatMap If a value is present, apply the provided Option-bearing mapping function to it, return that result,
// otherwise return an empty Option. This method is similar to Map(), but the provided mapper is one whose result is
// already an Option, and if invoked, FlatMap does not wrap it with an additional Option.
func (o Option[T]) FlatMap[U any](mapper func(T) Option[U]) Option[U] {
	if o.present {
		return mapper(o.value)
	}
	return NoneOf[U]()
}

// Fold allows to transform an Option[T] into value of another type U.
// Fold accepts two transformation functions as a parameter.
//
// onPresent function will be applied to a value of a non-empty Option[T]
// onAbsent function will be called if this Option is empty
//
// Both of the functions must return a value of type U and must be free of side effects
func (o Option[T]) Fold[U any](onPresent func(T) U, onAbsent func() U) U {
	if o.present {
		return onPresent(o.value)
	}
	return onAbsent()
}
