package asynctask_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func TestWaitAnyNoTask(t *testing.T) {
	t.Parallel()
	ctx, _ := newTestContextWithTimeout(t, 2*time.Second)

	err := asynctask.WaitAny(ctx, nil)
	assert.NoError(t, err)
}

func TestWaitAny(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 2*time.Second)

	start := time.Now()
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAny(ctx, nil, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	// should finish after right away
	assert.True(t, elapsed < 2*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	start = time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 = asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	err = asynctask.WaitAny(ctx, &asynctask.WaitAnyOptions{FailOnAnyError: true}, countingTsk1, countingTsk2, countingTsk3)
	elapsed = time.Since(start)
	assert.NoError(t, err)
	cancelTaskExecution()

	// should finish right after countingTsk3
	assert.True(t, elapsed >= 20*time.Millisecond && elapsed < 200*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAnyContextCancel(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 2*time.Second)

	start := time.Now()

	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	go func() {
		time.Sleep(5 * time.Millisecond)
		cancelTaskExecution()
	}()
	err := asynctask.WaitAny(ctx, nil, countingTsk1, countingTsk2)
	elapsed := time.Since(start)
	assert.Error(t, err)
	assert.Equal(t, "WaitAny context canceled", err.Error(), "expecting context canceled error")
	// should finish right after countingTsk3
	assert.True(t, elapsed >= 5*time.Millisecond && elapsed < 200*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAnyErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)
	err := asynctask.WaitAny(ctx, nil, errorTsk, completedTsk)
	assert.NoError(t, err)
	elapsed := time.Since(start)
	// should finish after right away
	assert.True(t, elapsed < 20*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))
	completedTskState := completedTsk.State()
	assert.Equal(t, asynctask.StateCompleted, completedTskState, "completed task should finished")

	start = time.Now()
	countingTsk := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	errorTsk = asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	err = asynctask.WaitAny(ctx, nil, countingTsk, errorTsk, panicTsk)
	// there is a succeed task
	assert.NoError(t, err)
	elapsed = time.Since(start)

	countingTskState := countingTsk.State()
	panicTskState := panicTsk.State()
	errTskState := errorTsk.State()
	cancelTaskExecution() // all assertion variable captured, cancel counting task

	// should only finish after longest task.
	assert.True(t, elapsed >= 40*10*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	assert.Equal(t, asynctask.StateCompleted, countingTskState, "countingTask should NOT finished")
	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should failed")
	assert.Equal(t, asynctask.StateFailed, panicTskState, "panic task should Not failed")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAnyAllFailCase(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	err := asynctask.WaitAny(ctx, nil, errorTsk, panicTsk)
	assert.Error(t, err)

	panicTskState := panicTsk.State()
	errTskState := errorTsk.State()
	elapsed := time.Since(start)
	cancelTaskExecution() // all assertion variable captured, cancel counting task

	assert.Equal(t, "expected error", err.Error(), "expecting first error")
	// should finsh after both error
	assert.True(t, elapsed >= 20*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should failed")
	assert.Equal(t, asynctask.StateFailed, panicTskState, "panic task should Not failed")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAnyErrorWithFailOnAnyErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)
	err := asynctask.WaitAny(ctx, &asynctask.WaitAnyOptions{FailOnAnyError: true}, errorTsk, completedTsk)
	assert.NoError(t, err)
	elapsed := time.Since(start)
	// should finish after right away
	assert.True(t, elapsed < 20*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	start = time.Now()
	countingTsk := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	errorTsk = asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	err = asynctask.WaitAny(ctx, &asynctask.WaitAnyOptions{FailOnAnyError: true}, countingTsk, errorTsk, panicTsk)
	assert.Error(t, err)
	completedTskState := completedTsk.State()
	assert.Equal(t, asynctask.StateCompleted, completedTskState, "completed task should finished")

	countingTskState := countingTsk.State()
	panicTskState := panicTsk.State()
	errTskState := errorTsk.State()
	elapsed = time.Since(start)
	cancelTaskExecution() // all assertion variable captured, cancel counting task

	assert.Equal(t, "expected error", err.Error(), "expecting first error")
	// should finsh after first error
	assert.True(t, elapsed >= 10*time.Millisecond && elapsed < 20*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	assert.Equal(t, asynctask.StateRunning, countingTskState, "countingTask should NOT finished")
	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should failed")
	assert.Equal(t, asynctask.StateRunning, panicTskState, "panic task should Not failed")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}
