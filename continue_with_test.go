package asynctask_test

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func getAdvancedCountingTask(countFrom, step int, sleepInterval time.Duration) asynctask.AsyncFunc {
	return func(ctx context.Context) (interface{}, error) {
		t := ctx.Value(testContextKey).(*testing.T)

		result := countFrom
		for i := 0; i < step; i++ {
			select {
			case <-time.After(sleepInterval):
				t.Logf("  working %d", i)
				result++
			case <-ctx.Done():
				t.Log("work canceled")
				return result, nil
			}
		}
		return result, nil
	}
}

func TestContinueWith(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	t1 := asynctask.Start(ctx, getAdvancedCountingTask(0, 10, 20*time.Millisecond))
	t2 := t1.ContinueWith(ctx, func(fCtx context.Context, input interface{}) (interface{}, error) {
		fromPrevTsk := input.(int)
		return getAdvancedCountingTask(fromPrevTsk, 10, 20*time.Millisecond)(fCtx)
	})
	t3 := t1.ContinueWith(ctx, func(fCtx context.Context, input interface{}) (interface{}, error) {
		fromPrevTsk := input.(int)
		return getAdvancedCountingTask(fromPrevTsk, 12, 20*time.Millisecond)(fCtx)
	})

	result, err := t2.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t2.State(), "Task should complete with no error")
	assert.Equal(t, result, 20)

	result, err = t3.Wait(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t3.State(), "Task should complete with no error")
	assert.Equal(t, result, 22)
}

func TestContinueWithFailureCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	t1 := asynctask.Start(ctx, getErrorTask("devide by 0", 10*time.Millisecond))
	t2 := t1.ContinueWith(ctx, func(fCtx context.Context, input interface{}) (interface{}, error) {
		fromPrevTsk := input.(int)
		return getAdvancedCountingTask(fromPrevTsk, 10, 20*time.Millisecond)(fCtx)
	})
	t3 := t1.ContinueWith(ctx, func(fCtx context.Context, input interface{}) (interface{}, error) {
		fromPrevTsk := input.(int)
		return getAdvancedCountingTask(fromPrevTsk, 12, 20*time.Millisecond)(fCtx)
	})

	_, err := t2.Wait(ctx)
	assert.Error(t, err)
	assert.Equal(t, asynctask.StateFailed, t2.State(), "Task2 should fail since preceeding task failed")
	assert.Equal(t, "devide by 0", err.Error())

	_, err = t3.Wait(ctx)
	assert.Error(t, err)
	assert.Equal(t, asynctask.StateFailed, t3.State(), "Task3 should fail since preceeding task failed")
	assert.Equal(t, "devide by 0", err.Error())
}
