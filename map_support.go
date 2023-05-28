package seqs

import "sync"

func sequentialMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	return ProducerSeq[U]{
		parallelism: input.Parallelism(),
		producer: func() Option[U] {
			v, ok := input.Next()
			if ok {
				u := mapper(v)
				return SomeOf(u)
			} else {
				return NoneOf[U]()
			}
		},
	}
}

func parallelMap[T any, U any](input Seq[T], mapper func(T) U) Seq[U] {
	output := make(chan U)
	allDone := make(chan bool)

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			v, ok := input.Next()
			for ok {
				u := mapper(v)
				output <- u
				v, ok = input.Next()
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		allDone <- true
	}()

	return ProducerSeq[U]{
		parallelism: input.Parallelism(),
		producer: func() Option[U] {
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
		},
	}
}

func sequentialFlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	chunkSeq := sequentialMap(input, mapper)
	chunk, chunkValid := chunkSeq.Next()
	return ProducerSeq[U]{
		parallelism: input.Parallelism(),
		producer: func() Option[U] {
			for {
				if !chunkValid {
					chunk, chunkValid = chunkSeq.Next()
					if !chunkValid {
						return NoneOf[U]()
					}
				}
				u, ok := chunk.Next()
				if ok {
					return SomeOf(u)
				} else {
					chunkValid = false
					continue
				}
			}
		},
	}
}

func parallelFlatMap[T any, U any](input Seq[T], mapper func(T) Seq[U]) Seq[U] {
	output := make(chan U)
	allDone := make(chan bool)

	var wg sync.WaitGroup
	for i := 0; i < input.Parallelism(); i++ {
		wg.Add(1)
		go func() {
			v, ok := input.Next()
			for ok {
				u := mapper(v)
				u1, ok1 := u.Next()
				for ok1 {
					output <- u1
					u1, ok1 = u.Next()
				}
				v, ok = input.Next()
			}
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		allDone <- true
	}()

	return ProducerSeq[U]{
		parallelism: input.Parallelism(),
		producer: func() Option[U] {
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
		},
	}
}
