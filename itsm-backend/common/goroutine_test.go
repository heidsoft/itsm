package common

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
)

func TestGoSafe_RecoversFromPanic(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	panicTriggered := false

	// Test that GoSafe recovers from panic
	GoSafe(func() {
		panic("test panic")
	}, GoSafeOptions{
		Logger:   logger,
		TaskName: "test-panic",
	})

	// Give goroutine time to execute
	time.Sleep(100 * time.Millisecond)

	// If we reach here, panic was recovered
	panicTriggered = true
	assert.True(t, panicTriggered, "GoSafe should recover from panic")
}

func TestGoSafe_ExecutesNormally(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	executed := false

	GoSafe(func() {
		executed = true
	}, GoSafeOptions{
		Logger:   logger,
		TaskName: "test-normal",
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "GoSafe should execute function normally")
}

func TestGoSafeWithoutOptions(t *testing.T) {
	executed := false

	GoSafe(func() {
		executed = true
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "GoSafe without options should work")
}

func TestGoSafeWithContext(t *testing.T) {
	logger := zaptest.NewLogger(t).Sugar()
	executed := false

	GoSafeWithContext(func() {
		executed = true
	}, logger, "test-with-context")

	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "GoSafeWithContext should work")
}

func TestMustGoSafe(t *testing.T) {
	executed := false

	MustGoSafe(func() {
		executed = true
	})

	time.Sleep(100 * time.Millisecond)
	assert.True(t, executed, "MustGoSafe should work")
}
