# seqs

The sequence processing library. Provides capabilities similar to that of Java Streams.

Sequences are implemented using generics and require Go 1.19 or later.

`Seq[T]` is a sequence of elements of type `T` that supports sequential and parallel aggregate operations.

Sequences support transformations, filtering, and grouping computations organized as a sequence pipeline. Pipeline 
consists of a source (a Producer function which can be backed up by slice or channel or anything else really), 
the intermediate operations, which transforms a sequence into another sequence and a terminal operation which 
produces the result or a side effect.

## Sources

Sequences sources are really a just generator function that is called repeatedly.

Helper functions are provided to create sequences from slices, channels, map keys or map values.

Note that it is possible to create never ending sequences.

Sequences can be sequential, in that case they iterate over their source maintaining the order of original, or
they can be parallel, then the order of elements in the sequence is not guaranteed.
                            
## Supported operations

Note that **seqs** is a work in progress and more operations will be added.

Intermediate operations:
 - Filter
 - Map
 - FlatMap
 - Take
 - Skip
 - Ordered
 - Parallel
 - Sequential

Terminal operations:
 - ToSlice
 - ToOrderedSlice
 - ToMap
 - ToMultiMap
 - ForEach
 - Count
 - Accumulate
 - Reduce
 - AllMatch
 - AnyMatch


As Go doesn't support declaring new generic types on a method of generic interface, like Java, some of the sequence
operations are implemented not as methods of `Seq` interface. 

For example in Java you can have 
```java
 interface Seq<T> {
  <R> Seq<R> map(Function<T,R> mapper);
}
```
you can't do that in Go, so Map is implemented as this:
```go
 type Seq[T any] interface {}

 func Map[T any, R any](source Seq[T], func mapper(t T) R) Seq[R] {...}
```
So instead of doing

```go
    s := SeqOf(1,2,3,4,5)
    mapped := s.
        Filter(func(t int) {
         return t % 2 == 0
        }).
        Map(func(t int) string {
          return strconv.itoa(t)
        })
```
you will have to write it as follows
```go
    s := SeqOf(1,2,3,4,5)
    mapped := Map[int, string](s.Filter(func(t int) {
      return t % 2 == 0
    }), func(t int) string {
      return strconv.itoa(t)
    })
```

Note that Map function is still lazy and still supports both parallel and sequential sequences. It is just a bit more
awkward to use, because you can't use fluent chains with it.
