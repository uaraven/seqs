// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"github.com/onsi/gomega"
	"strconv"
	"testing"
)

func TestOptionApply(t *testing.T) {
	g := gomega.NewWithT(t)

	o := SomeOf(10)
	o.Apply(func(value int) {
		g.Expect(value).To(gomega.Equal(10))
	}, func() {
		g.Fail("Should not be called")
	})
}

func TestEmptyOptionApply(t *testing.T) {
	g := gomega.NewWithT(t)

	o := NoneOf[int]()
	o.Apply(func(value int) {
		g.Fail("Should not be called")
	}, func() {
	})
}

func TestApplyOption(t *testing.T) {
	g := gomega.NewWithT(t)

	o := SomeOf(10)
	r := ApplyOption[int, string](o, func(value int) string {
		return strconv.Itoa(value)
	}, func() string {
		return "none"
	})
	g.Expect(r).To(gomega.Equal("10"))
}

func TestApplyEmptyOption(t *testing.T) {
	g := gomega.NewWithT(t)

	o := NoneOf[int]()
	r := ApplyOption[int, string](o, func(value int) string {
		return strconv.Itoa(value)
	}, func() string {
		return "none"
	})
	g.Expect(r).To(gomega.Equal("none"))
}

func TestMapOptionPresent(t *testing.T) {
	g := gomega.NewWithT(t)

	o := SomeOf(10)
	om := MapOption[int, string](o, func(t int) string {
		return strconv.Itoa(t)
	})

	g.Expect(om.IsPresent()).To(gomega.BeTrue())
	g.Expect(om.Value()).To(gomega.Equal("10"))
	g.Expect(om.OrElse("20")).To(gomega.Equal("10"))
}

func TestMapOptionNone(t *testing.T) {
	g := gomega.NewWithT(t)

	o := NoneOf[int]()
	om := MapOption[int, string](o, func(t int) string {
		return strconv.Itoa(t)
	})

	g.Expect(om.IsPresent()).To(gomega.BeFalse())
	g.Expect(om.OrElse("20")).To(gomega.Equal("20"))
	g.Expect(om.Value()).To(gomega.Equal(""))
}

func TestOptionSomeToSeq(t *testing.T) {
	g := gomega.NewWithT(t)

	o := SomeOf(10)
	s := o.ToSeq().ToSlice()

	g.Expect(s).To(gomega.HaveLen(1))
	g.Expect(s).To(gomega.ContainElements(10))
}

func TestOptionNoneToSeq(t *testing.T) {
	g := gomega.NewWithT(t)

	o := NoneOf[int]()
	s := o.ToSeq().ToSlice()

	g.Expect(s).To(gomega.HaveLen(0))
}
