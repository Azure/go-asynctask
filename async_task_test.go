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

func getCountingTask(sleepDuration time.Duration) asynctask.AsyncFunc {
	return func(ctx context.Context) (interface{}, error) {
		t := ctx.Value(testContextKey).(*testing.T)

		result := 0
		for i := 0; i < 10; i++ {
			select {
			case <-time.After(sleepDuration):
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
	t1 := asynctask.Start(ctx, getCountingTask(200*time.Millisecond))

	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	rawResult, err := t1.Wait()
	assert.NoError(t, err)

	assert.Equal(t, asynctask.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	result := rawResult.(int)
	assert.Equal(t, result, 9)

	//assert.Fail(t, "just want to see if trace is working")
}

func TestCancelFunc(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	t1 := asynctask.Start(ctx, getCountingTask(200*time.Millisecond))

	assert.Equal(t, asynctask.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(time.Second * 1)
	t1.Cancel()

	rawResult, err := t1.Wait()
	assert.NoError(t, err)

	assert.Equal(t, asynctask.StateCanceled, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	result := rawResult.(int)
	assert.Less(t, result, 9)

	// cancel a task shouldn't cancel it's parent context.
	select {
	case <-ctx.Done():
		assert.Fail(t, "parent context got canceled")
	default:
		t.Log("parent context still running")
	}
}

func TestCrazyCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	tasks := map[int]*asynctask.TaskStatus{}
	for i := 0; i < 10000; i++ {
		tasks[i] = asynctask.Start(ctx, getCountingTask(200*time.Millisecond))
	}

	time.Sleep(time.Second * 1)
	for i := 0; i < 10000; i += 2 {
		tasks[i].Cancel()
	}

	for i := 0; i < 10000; i += 2 {
		rawResult, err := tasks[i].Wait()
		assert.NoError(t, err)
		assert.NotNil(t, rawResult)

		result := rawResult.(int)
		if i%2 == 0 {
			assert.Less(t, result, 9)
		} else {
			assert.Equal(t, result, 9)
		}
	}
}
