package seqs

import (
	. "github.com/onsi/gomega"
	"runtime"
	"sort"
	"testing"
)

func TestSliceFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(1000)

	seq := NewSeq(NewSliceProducer(slice))
	nseq := seq.Filter(func(value int) bool {
		return value%2 == 0
	})
	result := nseq.ToSlice()

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	expected := make([]int, 0)
	for i := 0; i < len(slice); i++ {
		if i%2 == 0 {
			expected = append(expected, i)
		}
	}

	g.Expect(result).To(BeEquivalentTo(expected))
}

func TestSliceParallelFilter(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(1000)

	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	nseq := seq.Filter(func(value int) bool {
		return value%2 == 0
	})
	result := nseq.ToSlice()

	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})

	expected := make([]int, 0)
	for i := 0; i < len(slice); i++ {
		if i%2 == 0 {
			expected = append(expected, i)
		}
	}

	g.Expect(result).To(BeEquivalentTo(expected))
}

func TestMap(t *testing.T) {
	g := NewGomegaWithT(t)

	data := makeSlice(1000)
	seq := NewParallelSeq[int](NewSliceProducer(data), runtime.NumCPU())
	mapped := Map[int, int](seq, func(t int) int {
		return t * 2
	}).ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})

	expected := make([]int, len(data))
	for i := 0; i < len(data); i++ {
		expected[i] = data[i] * 2
	}

	g.Expect(mapped).To(BeEquivalentTo(expected))
}

func TestFlatMap(t *testing.T) {
	g := NewGomegaWithT(t)

	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{9},
	}
	seq := NewSeq[[]int](NewSliceProducer(superSlice))
	flatMap := FlatMap[[]int](seq, func(t []int) Seq[int] {
		return NewSeq[int](NewSliceProducer[int](t))
	})
	result := flatMap.ToSlice()
	g.Expect(result).To(BeEquivalentTo([]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}))
}

func TestFlatMapWithNilMap(t *testing.T) {
	g := NewGomegaWithT(t)

	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{},
		{9},
	}
	seq := NewSeq[[]int](NewSliceProducer(superSlice))
	flatMap := FlatMap[[]int](seq, func(t []int) Seq[int] {
		if len(t) == 0 {
			return nil
		}
		return NewSeq[int](NewSliceProducer[int](t))
	})
	result := flatMap.ToSlice()
	g.Expect(result).To(BeEquivalentTo([]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}))
}

func TestParallelFlatMap(t *testing.T) {
	g := NewGomegaWithT(t)

	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{9},
	}
	seq := NewParallelSeq[[]int](NewSliceProducer(superSlice), 4)
	flatMap := FlatMap[[]int](seq, func(t []int) Seq[int] {
		return NewSeq[int](NewSliceProducer[int](t))
	})
	result := flatMap.ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})
	g.Expect(result).To(BeEquivalentTo([]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}))
}

func TestParallelFlatMapWithNilMap(t *testing.T) {
	g := NewGomegaWithT(t)

	superSlice := [][]int{
		{0, 1, 2, 3},
		{4, 5, 6},
		{7, 8},
		{},
		{9},
	}
	seq := NewParallelSeq[[]int](NewSliceProducer(superSlice), 4)
	flatMap := FlatMap[[]int](seq, func(t []int) Seq[int] {
		if len(t) == 0 {
			return nil
		}
		return NewSeq[int](NewSliceProducer[int](t))
	})
	result := flatMap.ToSortedSlice(func(t1 int, t2 int) int {
		return t1 - t2
	})
	g.Expect(result).To(BeEquivalentTo([]int{
		0, 1, 2, 3, 4, 5, 6, 7, 8, 9,
	}))
}

func TestProducerSeq_First(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.First()
	g.Expect(v.IsPresent()).To(BeTrue())
	g.Expect(v.Value()).To(Equal(0))
	slice = makeSlice(0)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.First()
	g.Expect(v.IsPresent()).To(BeFalse())
}

func TestProducerSeq_Rest(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Rest().ToSlice()
	g.Expect(v).To(BeEquivalentTo([]int{1, 2, 3, 4}))
	slice = makeSlice(0)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.Rest().ToSlice()
	g.Expect(v).To(HaveLen(0))
}

func TestProducerSeq_Reduce(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(5)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Reduce(func(a int, b int) int {
		return a + b
	})

	g.Expect(v).To(Equal(10))

	slice = makeSlice(51)
	seq = NewParallelSeq(NewSliceProducer(slice), 4)
	v = seq.Reduce(func(a int, b int) int {
		return a + b
	})

	g.Expect(v).To(Equal(1275))
}

func TestProducerSeq_Accumulate(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := []string{"a", "bb", "ccc", "dddd", "eeeee"}
	seq := NewSeqFromSlice(slice, 4)
	v := Accumulate[string, int](seq, 0, func(a int, b string) int {
		return a + len(b)
	})

	g.Expect(v).To(Equal(15))
}

func TestProducerSeq_ReduceWithChannelProducer(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(51)
	src := make(chan int)
	seq := NewParallelSeq(NewChannelProducer(src), 4)

	go func() {
		for _, i := range slice {
			src <- i
		}
		close(src)
	}()

	v := seq.Reduce(func(a int, b int) int {
		return a + b
	})

	g.Expect(v).To(Equal(1275))
}

func TestNewSeqFromMapKeys(t *testing.T) {
	g := NewGomegaWithT(t)
	src := map[int]string{
		4: "four",
		2: "two",
		1: "one",
		3: "three",
	}
	keys := NewSeqFromMapKeys(src).ToSlice()
	g.Expect(keys).To(HaveLen(4))
	g.Expect(keys).To(ContainElements(1, 2, 3, 4))
}

func TestNewSeqFromMapValues(t *testing.T) {
	g := NewGomegaWithT(t)
	src := map[int]string{
		4: "four",
		2: "two",
		1: "one",
		3: "three",
	}
	vals := NewSeqFromMapValues(src).ToSlice()
	g.Expect(vals).To(HaveLen(4))
	g.Expect(vals).To(ContainElements("one", "two", "three", "four"))
}

func TestProducerSeq_Count(t *testing.T) {
	g := NewGomegaWithT(t)
	slice := makeSlice(51)
	seq := NewParallelSeq(NewSliceProducer(slice), 4)
	v := seq.Count()

	g.Expect(v).To(Equal(int64(51)))
}

func TestEmptySeq(t *testing.T) {
	g := NewGomegaWithT(t)
	r := EmptySeq[int]().ToSlice()

	g.Expect(r).To(HaveLen(0))
}

func TestProducerSeq_Peek(t *testing.T) {
	g := NewGomegaWithT(t)

	src := SeqOf(1, 2, 3, 4, 5)
	dst := make([]int, 0)

	src.Peek(func(value int) {
		dst = append(dst, value)
	}).Count()

	g.Expect(dst).To(HaveLen(5))
	g.Expect(dst).To(ContainElements(1, 2, 3, 4, 5))
}

func TestProducerSeq_Take(t *testing.T) {
	g := NewGomegaWithT(t)

	res1 := SeqOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Take(4).ToSlice()
	g.Expect(res1).To(ContainElements(1, 2, 3, 4))

	res2 := SeqOf(1, 2, 3, 4).Take(6).ToSlice()
	g.Expect(res2).To(ContainElements(1, 2, 3, 4))
}

func TestProducerSeq_Skip(t *testing.T) {
	g := NewGomegaWithT(t)

	res1 := SeqOf(1, 2, 3, 4, 5, 6, 7, 8, 9, 10).Skip(4).ToSlice()
	g.Expect(res1).To(ContainElements(5, 6, 7, 8, 9, 10))

	res2 := SeqOf(1, 2, 3, 4).Skip(6).ToSlice()
	g.Expect(res2).To(HaveLen(0))

	res3 := SeqOf(1, 2, 3, 4).Skip(4).ToSlice()
	g.Expect(res3).To(HaveLen(0))
}

func TestProducerSeq_Ordered(t *testing.T) {
	g := NewGomegaWithT(t)

	src := NewSeqFromSlice([]int{10, 2, 5, 9, 3, 7, 6, 8, 4, 1}, 4)
	res := src.Ordered(func(t1 int, t2 int) int {
		return t1 - t2
	}).ToSlice()

	g.Expect(res).To(Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}))
}

func TestProducerSeq_Parallel(t *testing.T) {
	g := NewGomegaWithT(t)

	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	src := NewSeqFromSlice(slice)

	g.Expect(src.ToSlice()).To(Equal(slice))

	psrc := src.Parallel(4)

	g.Expect(psrc.ToSlice()).ToNot(Equal(slice))
	g.Expect(psrc.Parallelism()).To(Equal(4))
}

func TestProducerSeq_Sequential(t *testing.T) {
	g := NewGomegaWithT(t)

	slice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	src := NewSeqFromSlice(slice, 4)

	ssrc := src.Sequential()
	g.Expect(ssrc.ToSlice()).To(ContainElements(slice))
	g.Expect(ssrc.Parallelism()).To(Equal(1))
}

func TestToMultiMap(t *testing.T) {
	g := NewGomegaWithT(t)
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 2)
	result := ToMultiMap[int, string, string](seq, func(e string) int { return len(e) }, func(e string) string { return e })

	g.Expect(result).To(HaveLen(5))
	g.Expect(result[1]).To(ContainElements([]string{"a", "i"}))
	g.Expect(result[2]).To(ContainElements([]string{"bb", "hh"}))
	g.Expect(result[3]).To(ContainElements([]string{"ccc", "ggg"}))
	g.Expect(result[4]).To(ContainElements([]string{"dddd", "ffff"}))
	g.Expect(result[5]).To(ContainElements([]string{"eeeee"}))
}

func TestToMap(t *testing.T) {
	g := NewGomegaWithT(t)
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)
	result := ToMap[int, string, string](seq,
		func(e string) int { return len(e) },
		func(e string) string { return e },
		func(k int, v1 string, v2 string) string {
			if v1 < v2 {
				return v1
			} else {
				return v2
			}
		})

	g.Expect(result).To(BeEquivalentTo(map[int]string{
		1: "a",
		2: "bb",
		3: "ccc",
		4: "dddd",
		5: "eeeee",
	}))
}

func TestProducerSeq_AnyMatch(t *testing.T) {
	g := NewGomegaWithT(t)
	seq := NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)

	result := seq.AnyMatch(func(value string) bool {
		return value == "ggg"
	})
	g.Expect(result).To(BeTrue())

	seq = NewSeqFromSlice([]string{"a", "bb", "ccc", "dddd", "eeeee", "ffff", "ggg", "hh", "i"}, 4)
	result = seq.AnyMatch(func(value string) bool {
		return value == "xyz"
	})
	g.Expect(result).To(BeFalse())
}

func TestProducerSeq_AllMatch(t *testing.T) {
	g := NewGomegaWithT(t)
	seq := NewSeqFromSlice(makeSlice(100), 4)

	result := seq.AllMatch(func(value int) bool {
		return value < 1000
	})
	g.Expect(result).To(BeTrue())

	seq = NewSeqFromSlice(makeSlice(100), 4)
	result = seq.AllMatch(func(value int) bool {
		return value != 50
	})
	g.Expect(result).To(BeFalse())
}

func makeSlice(n int) []int {
	slice := make([]int, n)
	for i := 0; i < len(slice); i++ {
		slice[i] = i
	}
	return slice
}

//func TestMapPerf(t *testing.T) {
//	testSequential := func() {
//		micros := int64(0)
//
//		data := makeSlice(10000)
//		for i := 0; i < 10; i++ {
//			start := time.Now()
//			seq := NewSeq[int](NewSliceProducer(data))
//			Map[int, int](seq, func(t int) int {
//				time.Sleep(time.Microsecond)
//				return t * 2
//			}).ToSlice()
//			elapsed := time.Now().Sub(start)
//			micros = micros + elapsed.Microseconds()
//		}
//		total := float64(micros) / 100.0
//		fmt.Printf("Sequential Map: %5.4f\n", total)
//	}
//
//	testParallel := func() {
//		micros := int64(0)
//
//		data := makeSlice(10000)
//		for i := 0; i < 10; i++ {
//			start := time.Now()
//			seq := NewParallelSeq[int](NewSliceProducer(data), 4)
//			Map[int, int](seq, func(t int) int {
//				time.Sleep(time.Microsecond)
//				return t * 2
//			}).ToSlice()
//			elapsed := time.Now().Sub(start)
//			micros = micros + elapsed.Microseconds()
//		}
//		total := float64(micros) / 100.0
//		fmt.Printf("Parallel Map, 4 goroutines: %5.4f\n", total)
//	}
//
//	testSequential()
//	testParallel()
//}
