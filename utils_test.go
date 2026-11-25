package seqs

import (
	"context"
	"github.com/stretchr/testify/assert"
	"os"
	"os/exec"
	"testing"
	"time"
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

// inSubprocess runs body in a child process (a re-execution of the test binary) and fails the test if the child
// panics, crashes or does not finish in time.
//
// Panics raised inside worker goroutines cannot be recovered and a livelocked worker keeps spinning forever. Running
// such scenarios in-process would kill or poison the whole test binary, so they are isolated here.
func inSubprocess(t *testing.T, timeout time.Duration, body func(t *testing.T)) {
	t.Helper()
	const envKey = "SEQS_TEST_SUBPROCESS"
	if os.Getenv(envKey) == t.Name() {
		body(t)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^"+t.Name()+"$", "-test.count=1")
	cmd.Env = append(os.Environ(), envKey+"="+t.Name())
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("scenario did not finish within %v (hang or livelock)", timeout)
	}
	if err != nil {
		if len(out) > 3000 {
			out = out[:3000]
		}
		t.Fatalf("scenario failed: %v\n%s", err, out)
	}
}
