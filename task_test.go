package asynctask_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

const testContextKey string = "testing"
const countingTaskDefaultStepLatency time.Duration = 20 * time.Millisecond

func newTestContext(t *testing.T) context.Context {
	return context.WithValue(context.TODO(), testContextKey, t)
}

func newTestContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithValue(context.TODO(), testContextKey, t), timeout)
}

func getCountingTask(countTo int, taskId string, sleepInterval time.Duration) asynctask.AsyncFunc[int] {
	return func(ctx context.Context) (*int, error) {
		t := ctx.Value(testContextKey).(*testing.T)

		result := 0
		for i := 0; i < countTo; i++ {
			select {
			case <-time.After(sleepInterval):
				t.Logf("[%s]: counting %d", taskId, i)
				result = i
			case <-ctx.Done():
				// testing.Logf would cause DataRace error when test is already finished: https://github.com/golang/go/issues/40343
				// leave minor time buffer before exit test to finish this last logging at least.
				t.Logf("[%s]: work canceled", taskId)
				return &result, nil
			}
		}
		return &result, nil
	}
}

func TestEasyGenericCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, "easyTask", countingTaskDefaultStepLatency))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	rawResult, err := t1.Result(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	assert.Equal(t, *rawResult, 9)

	// wait Again,
	start := time.Now()
	rawResult, err = t1.Result(ctx)
	elapsed := time.Since(start)
	// nothing should change
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	assert.Equal(t, *rawResult, 9)

	// 3 microSecond doesn't work anymore, the mutex lock does cost some time
	assert.True(t, elapsed.Microseconds() < 10, fmt.Sprintf("Second wait should have return immediately: %s", elapsed))
}

func TestCancelFuncOnGeneric(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, "cancelTask", countingTaskDefaultStepLatency))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(countingTaskDefaultStepLatency)
	t1.Cancel()

	_, err := t1.Result(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")

	// I can cancel again, and nothing changes
	time.Sleep(countingTaskDefaultStepLatency)
	t1.Cancel()
	_, err = t1.Result(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")

	// cancel a task shouldn't cancel it's parent context.
	select {
	case <-ctx.Done():
		assert.Fail(t, "parent context got canceled")
	default:
		t.Log("parent context still running")
	}
}

func TestConsistentResultAfterCancelGenericTask(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, "consistentTask1", countingTaskDefaultStepLatency))
	t2 := asynctask.Start(ctx, getCountingTask(10, "consistentTask2", countingTaskDefaultStepLatency))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(countingTaskDefaultStepLatency)
	start := time.Now()
	t1.Cancel()
	duration := time.Since(start)
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "t1 should turn to Canceled")
	assert.True(t, duration < 1*time.Millisecond, "cancel shouldn't take that long")

	// wait til routine finish
	rawResult, err := t2.Result(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t2.State(), "t2 should complete")
	assert.Equal(t, *rawResult, 9)

	// t1 should remain canceled and
	rawResult, err = t1.Result(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")
	assert.Equal(t, *rawResult, 0) // default value for int
}

func TestCompletedGenericTask(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	result := "something"
	tsk := asynctask.NewCompletedTask(&result)
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should in CompletedState")

	// nothing should happen
	tsk.Cancel()
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")

	// you get nil result and nil error
	resultGet, err := tsk.Result(ctx)
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")
	assert.NoError(t, err)
	assert.Equal(t, *resultGet, result)
}

func TestCrazyCaseGeneric(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	numOfTasks := 100 // if you have --race switch on: limit on 8128 simultaneously alive goroutines is exceeded, dying
	tasks := map[int]*asynctask.Task[int]{}
	for i := 0; i < numOfTasks; i++ {
		tasks[i] = asynctask.Start(ctx, getCountingTask(10, fmt.Sprintf("CrazyTask%d", i), countingTaskDefaultStepLatency))
	}

	// sleep 1 step, then cancel task with even number
	time.Sleep(countingTaskDefaultStepLatency)
	for i := 0; i < numOfTasks; i += 2 {
		tasks[i].Cancel()
	}

	time.Sleep(time.Duration(numOfTasks) * 2 * time.Microsecond)
	for i := 0; i < numOfTasks; i += 1 {
		rawResult, err := tasks[i].Result(ctx)

		if i%2 == 0 {
			assert.Equal(t, asynctask.ErrCanceled, err, fmt.Sprintf("task %s should be canceled, but it finished with %+v", fmt.Sprintf("CrazyTask%d", i), rawResult))
			assert.Equal(t, *rawResult, 0)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, *rawResult, 9)
		}
	}
}
