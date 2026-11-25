// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"runtime"
	"sync"
)

func sequentialMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	return NewParallelSeq(
		func() Option[U] {
			v := input.Next()
			return v.Map(mapper)
		}, input.Parallelism(),
	)
}

// lazyWorkers returns a producer that starts n worker goroutines on the first call and hands
// their results to the consumer.
//
// Each worker calls work, which pulls from the upstream and passes results to emit. emit returns
// false when the consumer has been abandoned, and work must then return.
//
// There is no Close API, so abandonment is detected by garbage collection: once the returned
// producer is unreachable, the workers are told to stop and the upstream is no longer read.
func lazyWorkers[U any](n int, work func(emit func(U) bool)) Producer[U] {
	type state struct{ once sync.Once }
	st := &state{}
	output := make(chan U)
	done := make(chan struct{})
	// The cleanup must not capture st (or anything referencing it), otherwise it never runs.
	runtime.AddCleanup(st, func(d chan struct{}) { close(d) }, done)

	emit := func(u U) bool {
		select {
		case output <- u:
			return true
		case <-done:
			return false
		}
	}
	start := func() {
		var wg sync.WaitGroup
		for range n {
			wg.Go(func() { work(emit) })
		}
		go func() {
			wg.Wait()
			close(output)
		}()
	}

	return func() Option[U] {
		st.once.Do(start)
		u, ok := <-output
		if !ok {
			return NoneOf[U]()
		}
		return SomeOf(u)
	}
}

func parallelMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	n := input.Parallelism()
	return NewParallelSeq(lazyWorkers(n, func(emit func(U) bool) {
		for v := input.Next(); v.IsPresent(); v = input.Next() {
			if !emit(mapper(v.Value())) {
				return
			}
		}
	}), n)
}

func sequentialFlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	chunkSeq := sequentialMap(input, mapper)
	chunk, chunkValid := chunkSeq.Next().Unwrap()
	return NewParallelSeq(
		func() Option[U] {
			for {
				if !chunkValid {
					chunk, chunkValid = chunkSeq.Next().Unwrap()
					if !chunkValid {
						return NoneOf[U]()
					}
				}
				if chunk.isZero() {
					chunkValid = false
					continue
				}
				u, ok := chunk.Next().Unwrap()
				if ok {
					return SomeOf(u)
				} else {
					chunkValid = false
					continue
				}
			}
		}, input.Parallelism(),
	)
}

func parallelFlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	n := input.Parallelism()
	return NewParallelSeq(lazyWorkers(n, func(emit func(U) bool) {
		for v := input.Next(); v.IsPresent(); v = input.Next() {
			u := mapper(v.Value())
			if u.isZero() {
				continue
			}
			for u1 := u.Next(); u1.IsPresent(); u1 = u.Next() {
				if !emit(u1.Value()) {
					return
				}
			}
		}
	}), n)
}
