// Copyright 2023 Les Voronin <me@ovoronin.info>
// SPDX-License-Identifier: MIT

package seqs

import (
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestOptionApply(t *testing.T) {
	o := SomeOf(10)
	o.Apply(func(value int) {
		assert.Equal(t, 10, value)
	}, func() {
		assert.Fail(t, "Should not be called")
	})
}

func TestEmptyOptionApply(t *testing.T) {
	o := NoneOf[int]()
	o.Apply(func(value int) {
		assert.Fail(t, "Should not be called")
	}, func() {
	})
}

func TestFoldOption(t *testing.T) {
	o := SomeOf(10)
	r := o.Fold(func(value int) string {
		return strconv.Itoa(value)
	}, func() string {
		return "none"
	})
	assert.Equal(t, "10", r)
}

func TestFoldEmptyOption(t *testing.T) {
	o := NoneOf[int]()
	r := o.Fold(func(value int) string {
		return strconv.Itoa(value)
	}, func() string {
		return "none"
	})
	assert.Equal(t, "none", r)
}

func TestMapOptionPresent(t *testing.T) {
	o := SomeOf(10)
	om := o.Map(func(t int) string {
		return strconv.Itoa(t)
	})

	assert.True(t, om.IsPresent())
	assert.Equal(t, "10", om.Value())
	assert.Equal(t, "10", om.OrElse("20"))
}

func TestMapOptionNone(t *testing.T) {
	o := NoneOf[int]()
	om := o.Map(func(t int) string {
		return strconv.Itoa(t)
	})

	assert.False(t, om.IsPresent())
	assert.Equal(t, "20", om.OrElse("20"))
	assert.Equal(t, "", om.Value())
}

func TestOptionSomeToSeq(t *testing.T) {
	o := SomeOf(10)
	s := o.ToSeq().ToSlice()

	assert.Len(t, s, 1)
	containsAll(t, s, 10)
}

func TestOptionNoneToSeq(t *testing.T) {
	o := NoneOf[int]()
	s := o.ToSeq().ToSlice()

	assert.Len(t, s, 0)
}
