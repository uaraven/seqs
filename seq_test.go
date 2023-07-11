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
