package asynctask_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func TestWaitAll(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 2*time.Second)
	defer cancelFunc()

	start := time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	// should only finish after longest task.
	assert.True(t, elapsed > 10*40*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))
}

func TestWaitAllFailFastCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	countingTsk := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk, errorTsk, panicTsk, completedTsk)
	countingTskState := countingTsk.State()
	panicTskState := countingTsk.State()
	elapsed := time.Since(start)

	cancelFunc() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error())
	// should fail before we finish panic task
	assert.True(t, elapsed.Milliseconds() < 20)

	// since we pass FailFast, countingTsk and panicTsk should be still running
	assert.Equal(t, asynctask.StateRunning, countingTskState)
	assert.Equal(t, asynctask.StateRunning, panicTskState)

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	start := time.Now()
	countingTsk := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: false}, countingTsk, errorTsk, panicTsk, completedTsk)
	countingTskState := countingTsk.State()
	panicTskState := panicTsk.State()
	errTskState := errorTsk.State()
	elapsed := time.Since(start)

	cancelFunc() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error(), "expecting first error")
	// should only finish after longest task.
	assert.True(t, elapsed > 10*40*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	assert.Equal(t, asynctask.StateCompleted, countingTskState, "countingTask should finished")
	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should failed")
	assert.Equal(t, asynctask.StateFailed, panicTskState, "panic task should failed")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllCanceledFailFast(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	start := time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	waitCtx, cancelFunc1 := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancelFunc1()

	elapsed := time.Since(start)
	err := asynctask.WaitAll(waitCtx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)

	cancelFunc() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	// should return before first task
	assert.True(t, elapsed < 10*2*time.Millisecond)

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllCanceled(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 4*time.Millisecond)
	defer cancelFunc()

	start := time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	waitCtx, cancelFunc1 := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancelFunc1()

	elapsed := time.Since(start)
	err := asynctask.WaitAll(waitCtx, &asynctask.WaitAllOptions{FailFast: false}, countingTsk1, countingTsk2, countingTsk3, completedTsk)

	cancelFunc() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	// should return before first task
	assert.True(t, elapsed < 10*2*time.Millisecond)

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllWithNoTasks(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 1*time.Millisecond)
	defer cancelFunc()

	err := asynctask.WaitAll(ctx, nil)
	assert.NoError(t, err)
}
