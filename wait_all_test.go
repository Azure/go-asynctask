package asynctask_test

import (
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func TestWaitAll(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, 2*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	start := time.Now()
	err := asynctask.WaitAll(countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	assert.True(t, elapsed > 10*200*time.Millisecond)
}

func TestWaitAllErrorCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	countingTsk := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	start := time.Now()
	err := asynctask.WaitAll(countingTsk, errorTsk, panicTsk, completedTsk)
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error())
	assert.True(t, elapsed.Milliseconds() < 15)
}
