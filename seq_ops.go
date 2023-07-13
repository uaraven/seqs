package seqs

import (
	"sync"
	"sync/atomic"
)

// Accumulate reduces a sequence to a single value, combining each element of the sequence with a previous value
// of the accumulator
//
// Initial value of the accumulator is passed as an argument
func Accumulate[T any, R any](input Seq[T], initial R, reducer func(a R, b T) R) R {
	output := make(chan T)

	var acc atomic.Value
	acc.Store(initial)
	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			v, ok := input.Next()
			for ok {
				for {
					accumValue := acc.Load()
					if acc.CompareAndSwap(accumValue, reducer(accumValue.(R), v)) {
						break
					}
				}
				v, ok = input.Next()
			}
		}()
	}
	wg.Wait()
	close(output)
	return acc.Load().(R)
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

// ToMap accumulates sequence elements into a map whose keys and values are the result of the applying the provided
// mapping functions to the input element.
// If the mapping keys contains duplicates then the value mapping function is applied to each element and the results
// are merged using the provided merging function
func ToMap[K comparable, V any, T any](input Seq[T], keyFunc func(T) K, valueFunc func(T) V, mergeFunc func(K, V, V) V) map[K]V {
	result := make(map[K]V)
	mapLock := sync.RWMutex{}

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, ok := input.Next()
			for ok {
				k := keyFunc(u)
				v := valueFunc(u)
				mapLock.Lock()
				if l, present := result[k]; present {
					result[k] = mergeFunc(k, l, v)
				} else {
					result[k] = v
				}
				mapLock.Unlock()
				u, ok = input.Next()
			}
		}()
	}

	// collector
	wg.Wait()
	return result
}

// MergeFirst is a merge function to be used with ToMap. It always selects the first value
func MergeFirst[K comparable, V any](key K, value1 V, value2 V) V {
	return value1
}

// MergeSecond is a merge function to be used with ToMap. It always selects the second value
func MergeSecond[K comparable, V any](key K, value1 V, value2 V) V {
	return value2
}

// ToMultiMap accumulates sequence elements into the map using the provider key and value mapping functions.
// Result is a map from the key to the slice of values. If multiple sequence elements map to the same
// key, the values will be appended to a slice mapped to the key.
func ToMultiMap[K comparable, V any, T any](input Seq[T], keyFunc func(T) K, valueFunc func(T) V) map[K][]V {
	result := make(map[K][]V)
	mapLock := sync.RWMutex{}

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, ok := input.Next()
			for ok {
				k := keyFunc(u)
				v := valueFunc(u)
				mapLock.Lock()
				if l, present := result[k]; present {
					result[k] = append(l, v)
				} else {
					result[k] = []V{v}
				}
				mapLock.Unlock()
				u, ok = input.Next()
			}
		}()
	}

	// collector
	wg.Wait()
	return result
}
