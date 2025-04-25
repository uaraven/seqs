package seqs

import (
	"github.com/onsi/gomega"
	"testing"
)

func TestNot(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	isOdd := func(v int) bool {
		return v%2 != 0
	}
	isEven := Not(isOdd)
	g.Expect(isEven(2)).To(gomega.BeTrue())
	g.Expect(isEven(1)).To(gomega.BeFalse())
}
