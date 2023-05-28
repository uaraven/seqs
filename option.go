package seqs

type Option[T any] interface {
	IsPresent() bool
	IfPresent(Consumer[T])

	Value() T
	OrElse(T) T

	Apply(onPresent Consumer[T], onAbsent func())
}

func SomeOf[T any](value T) *Some[T] {
	return &Some[T]{value: value}
}

func NoneOf[T any]() *None[T] {
	return &None[T]{}
}

func MapOption[T any, U any](option Option[T], mapper func(T) U) Option[U] {
	var result Option[U]
	option.Apply(func(value T) {
		result = SomeOf(mapper(value))
	}, func() {
		result = NoneOf[U]()
	})
	return result
}

func ApplyOption[T any, U any](option Option[T], onPresent func(T) U, onAbsent func() U) U {
	var result U
	option.Apply(func(value T) {
		result = onPresent(value)
	}, func() {
		result = onAbsent()
	})
	return result
}

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

func (s Some[T]) IfPresent(consumer Consumer[T]) {
	consumer(s.value)
}

func (s Some[T]) Apply(onPresent Consumer[T], _ func()) {
	onPresent(s.value)
}

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

func (n None[T]) IfPresent(Consumer[T]) {
	// do nothing
}

func (n None[T]) Apply(_ Consumer[T], onAbsent func()) {
	onAbsent()
}
