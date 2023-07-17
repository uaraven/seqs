package seqs

import "fmt"

// Option is a container that may or may not contain a value. If the value is present, then IsPresent() will return true
// and Value() will return the value
//
// Option can have two states: Some[T] and None[T]. Some contains a value while None is an Option without a value
type Option[T any] interface {
	// IsPresent returns true if Option contains a value
	IsPresent() bool
	// IfPresent accepts a Consumer function that will be executed if this Option has a value.
	// The value will be passed to the consumer
	//
	// If the Option is empty, consumer will not be called
	IfPresent(Consumer[T])

	// Value returns the value held in the Option.
	// Note: If Option is empty, the zero value of the type T will be returned
	Value() T
	// OrElse returns the value of the Option if it has one or the value provided as a parameter if the Option is empty
	OrElse(T) T
	// OrElseGet is similar to OrElse, but it uses a Provider function to generate a returned value of Option is empty
	OrElseGet(Provider[T]) T

	// Apply accepts two functions as a parameters. onPresent is called if the Option contains a value, and
	// onAbsent is called if the Option is empty
	Apply(onPresent Consumer[T], onAbsent func())

	// ToSeq converts the Option to a Seq. Resulting Seq will either contain one value, in case of Option containing
	//a value, or it will be an empty Seq, in case the Option is empty
	ToSeq() Seq[T]

	// Unwrap this Option into two values - the value contained in the Option (or zero value of type T) and the boolean
	// indicating whether the Option was not empty
	Unwrap() (T, bool)
}

// SomeOf creates a new Option[T] wrapping a provided value
func SomeOf[T any](value T) *Some[T] {
	return &Some[T]{value: value}
}

// NoneOf creates a new empty Option[T]
func NoneOf[T any]() *None[T] {
	return &None[T]{}
}

// MapOption performs a map operation on the Option. If the provided option parameter contains an
// non-empty Option, then mapper function is called on the Option value and new Option of the different type
// is returned
//
// If the provided option is empty then new empty Option[U] is returned
func MapOption[T any, U any](option Option[T], mapper func(T) U) Option[U] {
	return ApplyOption(option, func(v T) Option[U] {
		return SomeOf(mapper(v))
	}, func() Option[U] {
		return NoneOf[U]()
	})
}

// FlatMapOption If a value is present, apply the provided Option-bearing mapping function to it, return that result,
// otherwise return an empty Option. This method is similar to Map(), but the provided mapper is one whose result is
// already an Option, and if invoked, FlatMap does not wrap it with an additional Optional.
func FlatMapOption[T any, U any](option Option[T], mapper func(T) Option[U]) Option[U] {
	return ApplyOption[T, Option[U]](option, func(value T) Option[U] {
		return mapper(value)
	}, func() Option[U] {
		return NoneOf[U]()
	})
}

// ApplyOption allows to transform an Option[T] into value of another type U.
// ApplyOption accepts an option and two transformation functions as a parameter.
//
// onPresent function will be applied to a value of a non-empty Option[T]
// onAbsent function will be called if the option argument is an empty Option[T]
//
// Both of the functions must return a value of type U and must be free of side effects
func ApplyOption[T any, U any](option Option[T], onPresent func(T) U, onAbsent func() U) U {
	var result U
	option.Apply(func(value T) {
		result = onPresent(value)
	}, func() {
		result = onAbsent()
	})
	return result
}

// Some is an implementation of Option that contains a value
//
// Create new Some[T] using SomeOf() function
type Some[T any] struct {
	Option[T]
	value T
}

func (s Some[T]) IsPresent() bool {
	return true
}

func (s Some[T]) Value() T {
	return s.value
}

func (s Some[T]) OrElse(v T) T {
	return s.value
}

func (s Some[T]) OrElseGet(_ Provider[T]) T {
	return s.value
}

func (s Some[T]) IfPresent(consumer Consumer[T]) {
	consumer(s.value)
}

func (s Some[T]) Apply(onPresent Consumer[T], _ func()) {
	onPresent(s.value)
}

func (s Some[T]) ToSeq() Seq[T] {
	return SeqOf(s.value)
}

func (s Some[T]) Unwrap() (T, bool) {
	return s.value, true
}

func (s Some[T]) String() string {
	return fmt.Sprintf("Some[%v]", s.value)
}

// None is an implementation of Option that contains no value
//
// Create new None[T] using NoneOf() function
type None[T any] struct {
	Option[T]
}

func (n None[T]) IsPresent() bool {
	return false
}

func (n None[T]) Value() T {
	var zero T
	return zero
}

func (n None[T]) OrElse(v T) T {
	return v
}

func (n None[T]) OrElseGet(provider Provider[T]) T {
	return provider()
}

func (n None[T]) IfPresent(Consumer[T]) {
	// do nothing
}

func (n None[T]) Apply(_ Consumer[T], onAbsent func()) {
	onAbsent()
}

func (n None[T]) ToSeq() Seq[T] {
	return EmptySeq[T]()
}

func (n None[T]) Unwrap() (T, bool) {
	return n.Value(), false
}

func (n None[T]) String() string {
	return "None[]"
}
