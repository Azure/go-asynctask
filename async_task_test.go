package asynctask_test

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

type notMatter string

const testContextKey notMatter = "testing"

func newTestContext(t *testing.T) context.Context {
	return context.WithValue(context.TODO(), testContextKey, t)
}

func newTestContextWithTimeout(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithValue(context.TODO(), testContextKey, t), timeout)
}

func getCountingTask(countTo int, sleepInterval time.Duration) asynctask.AsyncFunc {
	return func(ctx context.Context) (interface{}, error) {
		t := ctx.Value(testContextKey).(*testing.T)

		result := 0
		for i := 0; i < countTo; i++ {
			select {
			case <-time.After(sleepInterval):
				t.Logf("  working %d", i)
				result = i
			case <-ctx.Done():
				t.Log("work canceled")
				return result, nil
			}
		}
		return result, nil
	}
}

func TestEasyCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	rawResult, err := t1.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	result := rawResult.(int)
	assert.Equal(t, result, 9)

	// wait Again,
	start := time.Now()
	rawResult, err = t1.Wait(ctx)
	elapsed := time.Since(start)
	// nothing should change
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	result = rawResult.(int)
	assert.Equal(t, result, 9)

	assert.True(t, elapsed.Microseconds() < 2, "Second wait should return immediately")
}

func TestCancelFunc(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(time.Second * 1)
	t1.Cancel()

	rawResult, err := t1.Wait(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")
	assert.Nil(t, rawResult)

	// I can cancel again, and nothing changes
	time.Sleep(time.Second * 1)
	t1.Cancel()
	rawResult, err = t1.Wait(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")
	assert.Nil(t, rawResult)

	// cancel a task shouldn't cancel it's parent context.
	select {
	case <-ctx.Done():
		assert.Fail(t, "parent context got canceled")
	default:
		t.Log("parent context still running")
	}
}

func TestConsistentResultAfterCancel(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	t1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	t2 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(time.Second * 1)
	start := time.Now()
	t1.Cancel()
	duration := time.Since(start)
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "t1 should turn to Canceled")
	assert.True(t, duration < 1*time.Millisecond, "cancel shouldn't take that long")

	// wait til routine finish
	rawResult, err := t2.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t2.State(), "t2 should complete")
	assert.Equal(t, rawResult, 9)

	// t1 should remain canceled and
	rawResult, err = t1.Wait(ctx)
	assert.Equal(t, asynctask.ErrCanceled, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should remain in cancel state")
	assert.Nil(t, rawResult)
}

func TestCompletedTask(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	tsk := asynctask.NewCompletedTask()
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should in CompletedState")

	// nothing should happen
	tsk.Cancel()
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")

	// you get nil result and nil error
	result, err := tsk.Wait(ctx)
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCrazyCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	numOfTasks := 8000 // if you have --race switch on: limit on 8128 simultaneously alive goroutines is exceeded, dying
	tasks := map[int]*asynctask.TaskStatus{}
	for i := 0; i < numOfTasks; i++ {
		tasks[i] = asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	}

	time.Sleep(200 * time.Millisecond)
	for i := 0; i < numOfTasks; i += 2 {
		tasks[i].Cancel()
	}

	for i := 0; i < numOfTasks; i += 1 {
		rawResult, err := tasks[i].Wait(ctx)

		if i%2 == 0 {
			assert.Equal(t, asynctask.ErrCanceled, err, "should be canceled")
			assert.Nil(t, rawResult)
		} else {
			assert.NoError(t, err)
			assert.NotNil(t, rawResult)

			result := rawResult.(int)
			assert.Equal(t, result, 9)
		}
	}
}
