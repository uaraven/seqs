package seqs

import "sync"

// Fold performs a fold operation on the sequence. It takes three functions as arguments:
// - zero: a function that returns the zero value for the fold result.
// - reducer: a function that takes an accumulator of type R and an element of type T, and returns a new accumulator of type R.
// - merge: a function that takes two accumulators of type R and merges them into a single accumulator of type R.
func (t Seq[T]) Fold[R any](
	zero func() R,
	reducer func(R, T) R,
	merge func(R, R) R) R {
	type workRec struct {
		acc R
		err any
	}
	if t.isZero() {
		return zero()
	}
	step := func(w *workRec, v T) (success bool) {
		defer func() {
			if r := recover(); r != nil {
				w.err = r
				success = false
			}
		}()
		w.acc = reducer(w.acc, v)
		return true
	}

	work := make([]workRec, t.Parallelism())
	for i := range work {
		work[i].acc = zero()
	}
	var wg sync.WaitGroup
	for i := range t.Parallelism() {
		wg.Go(func() {
			for v := t.Next(); v.IsPresent(); v = t.Next() {
				if !step(&work[i], v.Value()) {
					break
				}
			}
		})
	}
	wg.Wait()
	for _, w := range work {
		if w.err != nil {
			panic(w.err)
		}
	}
	result := work[0].acc
	for _, w := range work[1:] {
		result = merge(result, w.acc)
	}
	return result
}
