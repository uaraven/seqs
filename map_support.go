package seqs

import "sync"

func sequentialMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	return NewParallelSeq[U](
		func() Option[U] {
			v := input.Next()
			return MapOption(v, mapper)
		}, input.Parallelism(),
	)
}

func parallelMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	output := make(chan U)
	allDone := make(chan bool)

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			v := input.Next()
			for v.IsPresent() {
				u := mapper(v.Value())
				output <- u
				v = input.Next()
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		allDone <- true
	}()

	return NewParallelSeq(
		func() Option[U] {
			for {
				select {
				case <-allDone:
					close(output)
					return NoneOf[U]()
				case u, ok := <-output:
					if ok {
						return SomeOf(u)
					} else {
						return NoneOf[U]()
					}
				}
			}
		}, input.Parallelism(),
	)
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
				if chunk == nil {
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
	output := make(chan U)
	allDone := make(chan bool)

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			v := input.Next()
			for v.IsPresent() {
				u := mapper(v.Value())
				if u != nil {
					u1 := u.Next()
					for u1.IsPresent() {
						output <- u1.Value()
						u1 = u.Next()
					}
				}
				v = input.Next()
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		allDone <- true
	}()

	return NewParallelSeq(func() Option[U] {
		for {
			select {
			case <-allDone:
				close(output)
				return NoneOf[U]()
			case u, ok := <-output:
				if ok {
					return SomeOf(u)
				} else {
					return NoneOf[U]()
				}
			}
		}
	}, input.Parallelism())
}
