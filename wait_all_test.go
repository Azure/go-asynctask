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
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 2*time.Second)

	start := time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	assert.NoError(t, err)
	cancelTaskExecution()

	// should only finish after longest task.
	assert.True(t, elapsed > 10*40*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))
}

func TestWaitAllFailFastCase(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	countingTsk := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	errorTsk := asynctask.Start(ctx, getErrorTask("expected error", 10*time.Millisecond))
	panicTsk := asynctask.Start(ctx, getPanicTask(20*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	err := asynctask.WaitAll(ctx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk, errorTsk, panicTsk, completedTsk)
	countingTskState := countingTsk.State()
	panicTskState := countingTsk.State()
	errTskState := errorTsk.State()
	elapsed := time.Since(start)

	cancelTaskExecution() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error())
	// should fail before we finish panic task
	assert.True(t, elapsed.Milliseconds() < 20)

	// since we pass FailFast, countingTsk and panicTsk should be still running
	assert.Equal(t, asynctask.StateRunning, countingTskState)
	assert.Equal(t, asynctask.StateRunning, panicTskState)
	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should the one failed the waitAll.")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

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
	completedTskState := completedTsk.State()
	elapsed := time.Since(start)

	cancelTaskExecution() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.Equal(t, "expected error", err.Error(), "expecting first error")
	// should only finish after longest task.
	assert.True(t, elapsed > 10*40*time.Millisecond, fmt.Sprintf("actually elapsed: %v", elapsed))

	assert.Equal(t, asynctask.StateCompleted, countingTskState, "countingTask should finished")
	assert.Equal(t, asynctask.StateFailed, errTskState, "error task should failed")
	assert.Equal(t, asynctask.StateFailed, panicTskState, "panic task should failed")
	assert.Equal(t, asynctask.StateCompleted, completedTskState, "completed task should finished")

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllFailFastCancelingWait(t *testing.T) {
	t.Parallel()
	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 3*time.Second)

	start := time.Now()
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	waitCtx, cancelWait := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancelWait()

	err := asynctask.WaitAll(waitCtx, &asynctask.WaitAllOptions{FailFast: true}, countingTsk1, countingTsk2, countingTsk3, completedTsk)
	elapsed := time.Since(start)
	countingTsk1State := countingTsk1.State()
	countingTsk2State := countingTsk2.State()
	countingTsk3State := countingTsk3.State()
	completedTskState := completedTsk.State()
	cancelTaskExecution() // all assertion variable captured, cancel task execution

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	// should return before first task
	assert.True(t, elapsed < 10*2*time.Millisecond)
	assert.Equal(t, countingTsk1State, asynctask.StateRunning)
	assert.Equal(t, countingTsk2State, asynctask.StateRunning)
	assert.Equal(t, countingTsk3State, asynctask.StateRunning)
	assert.Equal(t, completedTskState, asynctask.StateCompleted)

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(1 * time.Millisecond)
}

func TestWaitAllCancelingWait(t *testing.T) {
	t.Parallel()

	ctx, cancelTaskExecution := newTestContextWithTimeout(t, 4*time.Millisecond)

	start := time.Now()
	rcCtx, rcCancel := context.WithCancel(context.Background())
	uncontrollableTask := asynctask.Start(ctx, getUncontrollableTask(rcCtx, t))
	countingTsk1 := asynctask.Start(ctx, getCountingTask(10, "countingPer40ms", 40*time.Millisecond))
	countingTsk2 := asynctask.Start(ctx, getCountingTask(10, "countingPer20ms", 20*time.Millisecond))
	countingTsk3 := asynctask.Start(ctx, getCountingTask(10, "countingPer2ms", 2*time.Millisecond))
	result := "something"
	completedTsk := asynctask.NewCompletedTask(&result)

	waitCtx, cancelWait := context.WithTimeout(ctx, 5*time.Millisecond)
	defer cancelWait()

	err := asynctask.WaitAll(waitCtx, &asynctask.WaitAllOptions{FailFast: false}, countingTsk1, countingTsk2, countingTsk3, completedTsk, uncontrollableTask)
	elapsed := time.Since(start)
	t.Logf("WaitAll finished, elapsed: %v", elapsed)
	cancelTaskExecution() // all assertion variable captured, cancel counting task

	assert.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded))
	// should return before first task
	assert.True(t, elapsed < 10*2*time.Millisecond)

	// cancel the remote control context to stop the uncontrollable task, or goleak.VerifyNone will fail.
	defer rcCancel()

	// counting task do testing.Logf in another go routine
	// while testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
	// wait minor time for the go routine to finish.
	time.Sleep(50 * time.Millisecond)
}

func TestWaitAllWithNoTasks(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 1*time.Millisecond)
	defer cancelFunc()

	err := asynctask.WaitAll(ctx, nil)
	assert.NoError(t, err)
}

// getUncontrollableTask return a task that is not honor context, it only hornor the remoteControl context.
func getUncontrollableTask(rcCtx context.Context, t *testing.T) asynctask.AsyncFunc[int] {
	return func(ctx context.Context) (*int, error) {
		for {
			select {
			case <-time.After(1 * time.Millisecond):
				if err := ctx.Err(); err != nil {
					t.Logf("[UncontrollableTask]: context %s, but not honoring it.", err)
				}
			case <-rcCtx.Done():
				t.Logf("[UncontrollableTask]: cancelled by remote control")
				return nil, rcCtx.Err()
			}
		}
	}
}
