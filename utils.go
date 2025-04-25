package seqs

// Not returns a new predicate that inverts the provided predicate function
func Not[T any](p Predicate[T]) Predicate[T] {
	return func(value T) bool {
		return !p(value)
	}
}
