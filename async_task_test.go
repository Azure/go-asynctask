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
	ctx := newTestContext(t)
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
	ctx := newTestContext(t)
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
	ctx := newTestContext(t)
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

func TestConsistentResultAfterTimeout(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	// t1 wait with short time, expect a timeout
	t1 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))
	// t2 wait with enough time, expect to complete
	t2 := asynctask.Start(ctx, getCountingTask(10, 200*time.Millisecond))

	assert.Equal(t, asynctask.StateRunning, t1.State(), "t1 should queued to Running")
	assert.Equal(t, asynctask.StateRunning, t2.State(), "t2 should queued to Running")

	rawResult, err := t1.WaitWithTimeout(ctx, 1*time.Second)
	assert.Equal(t, asynctask.ErrTimeout, err, "should return reason of error")
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "t1 get canceled after timeout")
	assert.Nil(t, rawResult, "didn't expect resule on canceled task")

	rawResult, err = t2.WaitWithTimeout(ctx, 2100*time.Millisecond)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t2.State(), "t2 should complete")
	assert.Equal(t, rawResult, 9)

	// even the t1 go routine finished, it shouldn't change any state
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "t1 should remain canceled even after routine finish")
	rawResult, err = t1.Wait(ctx)
	assert.Equal(t, asynctask.ErrTimeout, err, "should return reason of error")
	assert.Nil(t, rawResult, "didn't expect resule on canceled task")

	// invoke WaitWithTimeout again,
	start := time.Now()
	rawResult, err = t1.WaitWithTimeout(ctx, 1*time.Second)
	elapsed := time.Since(start)
	assert.Equal(t, asynctask.StateCanceled, t1.State(), "t1 should remain canceled even after routine finish")
	assert.Equal(t, asynctask.ErrTimeout, err, "should return reason of error")
	assert.Nil(t, rawResult, "didn't expect resule on canceled task")
	assert.True(t, elapsed.Microseconds() < 2, "Second wait should return immediately")
}

func TestCompletedTask(t *testing.T) {
	t.Parallel()

	tsk := asynctask.NewCompletedTask()
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should in CompletedState")

	// nothing should happen
	tsk.Cancel()
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")

	// you get nil result and nil error
	result, err := tsk.Wait(context.TODO())
	assert.Equal(t, asynctask.StateCompleted, tsk.State(), "Task should still in CompletedState")
	assert.NoError(t, err)
	assert.Nil(t, result)
}

func TestCrazyCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	numOfTasks := 10000
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
