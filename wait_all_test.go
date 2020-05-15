package asynctask_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func TestWaitAll(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, 2*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	start := time.Now()
	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	// should only finish after longest task.
	assert.True(t, elapsed > 10*200*time.Millisecond)
}

func TestWaitAllFailFastCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	countingTsk := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	start := time.Now()
	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk, errorTsk, panicTsk, completedTsk)
	countingTskState := countingTsk.State()
	panicTskState := countingTsk.State()
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error())
	// should fail before we finish panic task
	assert.True(t, elapsed.Milliseconds() < 15)

	// since we pass FailFast, countingTsk and panicTsk should be still running
	assert.Equal(t, asynctask.StateRunning, countingTskState)
	assert.Equal(t, asynctask.StateRunning, panicTskState)
}

func TestWaitAllErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	countingTsk := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	start := time.Now()
	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: false}, countingTsk, errorTsk, panicTsk, completedTsk)
	countingTskState := countingTsk.State()
	panicTskState := panicTsk.State()
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error())
	// should only finish after longest task.
	assert.True(t, elapsed > 10*200*time.Millisecond)

	// since we pass FailFast, countingTsk and panicTsk should be still running
	assert.Equal(t, asynctask.StateCompleted, countingTskState, "countingTask should finished")
	assert.Equal(t, asynctask.StateFailed, panicTskState, "panic task should failed")
}

func TestWaitAllCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, 2*time.Millisecond))
	completedTsk := asynctask.NewCompletedTask()

	waitCtx, cancelFunc1 := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancelFunc1()

	start := time.Now()
	err := asynctask.WaitAll(waitCtx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.Error(t, err)
	t.Log(err.Error())
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	// should return before first task
	assert.True(t, elapsed < 10*2*time.Millisecond)
}
