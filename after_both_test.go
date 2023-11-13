package asynctask_test

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func summarize2CountingTask(ctx context.Context, result1, result2 int) (int, error) {
	t := ctx.Value(testContextKey).(*testing.T)
	t.Logf("result1: %d", result1)
	t.Logf("result2: %d", result2)
	sum := result1 + result2
	t.Logf("sum: %d", sum)
	return sum, nil
}

func TestAfterBoth(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	t1 := asynctask.Start(ctx, getCountingTask(10, "afterboth.P1", 20*time.Millisecond))
	t2 := asynctask.Start(ctx, getCountingTask(10, "afterboth.P2", 20*time.Millisecond))
	t3 := asynctask.AfterBoth(ctx, t1, t2, summarize2CountingTask)

	sum, err := t3.Result(ctx)
	assert.NoError(t, err)
	assert.Equal(t, asynctask.StateCompleted, t3.State(), "Task should complete with no error")
	assert.Equal(t, sum, 18, "Sum should be 18")
}

func TestAfterBothFailureCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	errTask := asynctask.Start(ctx, getErrorTask("devide by 0", 10*time.Millisecond))
	countingTask := asynctask.Start(ctx, getCountingTask(10, "afterboth.P1", 20*time.Millisecond))

	task1Err := asynctask.AfterBoth(ctx, errTask, countingTask, summarize2CountingTask)
	_, err := task1Err.Result(ctx)
	assert.Error(t, err)

	task2Err := asynctask.AfterBoth(ctx, errTask, countingTask, summarize2CountingTask)
	_, err = task2Err.Result(ctx)
	assert.Error(t, err)

	task3NoErr := asynctask.AfterBoth(ctx, countingTask, countingTask, summarize2CountingTask)
	_, err = task3NoErr.Result(ctx)
	assert.NoError(t, err)
}

func TestAfterBothActionToFunc(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)

	countingTask1 := asynctask.Start(ctx, getCountingTask(10, "afterboth.P1", 20*time.Millisecond))
	countingTask2 := asynctask.Start(ctx, getCountingTask(10, "afterboth.P2", 20*time.Millisecond))
	t2 := asynctask.AfterBoth(ctx, countingTask1, countingTask2, asynctask.AfterBothActionToFunc(func(ctx context.Context, result1, result2 int) error {
		t := ctx.Value(testContextKey).(*testing.T)
		t.Logf("result1: %d", result1)
		t.Logf("result2: %d", result2)
		sum := result1 + result2
		t.Logf("sum: %d", sum)
		return nil
	}))
	_, err := t2.Result(ctx)
	assert.NoError(t, err)
}
