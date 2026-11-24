package seqs

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNot(t *testing.T) {
	isOdd := func(v int) bool {
		return v%2 != 0
	}
	isEven := Not(isOdd)
	assert.True(t, isEven(2))
	assert.False(t, isEven(1))
}

// containsAll asserts that actual contains all the expected elements
func containsAll[T any](t *testing.T, actual []T, expected ...T) {
	t.Helper()
	assert.Subset(t, actual, expected)
}
