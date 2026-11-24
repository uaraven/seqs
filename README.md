# seqs (v2)

```
go get github.com/uaraven/seqs/v2
```

The sequence processing library. Provides capabilities similar to that of Java Streams.

Sequences are implemented using generics and require Go 1.27 or later (generic methods are used).

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


`Seq[T]` and `Option[T]` are concrete types, so operations that introduce a new type parameter (`Map`, `FlatMap`,
`Accumulate`, `ToMap`, `ToMultiMap`, `Option.Map`, `Option.FlatMap`, `Option.Fold`) are ordinary generic methods
and can be chained:
```go
    s := SeqOf(1, 2, 3, 4, 5).
        Filter(func(t int) bool { return t%2 == 0 }).
        Map(func(t int) string { return strconv.Itoa(t) }).
        ToSlice()
```

Map is lazy and supports both parallel and sequential sequences.

## Migrating from v1

 - Import path is `github.com/uaraven/seqs/v2`.
 - `Seq` is a struct instead of an interface; `ProducerSeq` is gone. The zero value of `Seq` is an empty sequence
   (use `Seq[T]{}` instead of `nil` in `FlatMap` mappers).
 - `Option` is a struct instead of an interface; the `Some` and `None` types are gone.
 - `Map(seq, f)` → `seq.Map(f)`, `FlatMap(seq, f)` → `seq.FlatMap(f)`, `Accumulate(seq, init, f)` →
   `seq.Accumulate(init, f)`, `ToMap(seq, ...)` → `seq.ToMap(...)`, `ToMultiMap(seq, ...)` → `seq.ToMultiMap(...)`.
 - `MapOption(o, f)` → `o.Map(f)`, `FlatMapOption(o, f)` → `o.FlatMap(f)`, `ApplyOption(o, f, g)` → `o.Fold(f, g)`.
 - `golang.org/x/exp` is no longer a dependency.
