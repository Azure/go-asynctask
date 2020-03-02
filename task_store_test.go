package taskstore_test

import (
	"context"
	"testing"
	"time"

	taskstore "github.com/haitch/taskStore"
	"github.com/stretchr/testify/assert"
)

type notMatter string

const testContextKey notMatter = "testing"

func countingTask(ctx context.Context) (interface{}, error) {
	t := ctx.Value(testContextKey).(*testing.T)

	result := 0
	for i := 0; i < 10; i++ {
		select {
		case <-time.After(200 * time.Millisecond):
			t.Logf("  working %d", i)
			result = i
		case <-ctx.Done():
			t.Log("work canceled")
			return result, nil
		}
	}
	return result, nil
}

func TestEasyCase(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.TODO(), testContextKey, t)
	t1 := taskstore.StartTask(ctx, countingTask)

	assert.Equal(t, taskstore.StateRunning, t1.State(), "Task should queued to Running")

	rawResult, err := t1.Wait()
	assert.NoError(t, err)

	assert.Equal(t, taskstore.StateCompleted, t1.State(), "Task should complete by now")
	assert.NotNil(t, rawResult)
	result := rawResult.(int)
	assert.Equal(t, result, 9)

	//assert.Fail(t, "just want to see if trace is working")
}

func TestCancelFunc(t *testing.T) {
	t.Parallel()
	ctx := context.WithValue(context.TODO(), testContextKey, t)
	t1 := taskstore.StartTask(ctx, countingTask)

	assert.Equal(t, taskstore.StateRunning, t1.State(), "Task should queued to Running")

	time.Sleep(time.Second * 1)
	t1.Cancel()

	rawResult, err := t1.Wait()
	assert.NoError(t, err)

	assert.Equal(t, taskstore.StateCanceled, t1.State(), "Task should complete by now")
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
	ctx := context.WithValue(context.TODO(), testContextKey, t)
	tasks := map[int]*taskstore.TaskStatus{}
	for i := 0; i < 10000; i++ {
		tasks[i] = taskstore.StartTask(ctx, countingTask)
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
