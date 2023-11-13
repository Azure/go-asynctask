package asynctask_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func getPanicTask(sleepDuration time.Duration) asynctask.AsyncFunc[string] {
	return func(ctx context.Context) (string, error) {
		time.Sleep(sleepDuration)
		panic("yo")
	}
}

func getErrorTask(errorString string, sleepDuration time.Duration) asynctask.AsyncFunc[int] {
	return func(ctx context.Context) (int, error) {
		time.Sleep(sleepDuration)
		return 0, errors.New(errorString)
	}
}

func TestTimeoutCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	tsk := asynctask.Start(ctx, getCountingTask(10, "timeoutCase", countingTaskDefaultStepLatency))
	_, err := tsk.WaitWithTimeout(ctx, 30*time.Millisecond)
	assert.True(t, errors.Is(err, context.DeadlineExceeded), "expecting DeadlineExceeded")

	// the last Wait error should affect running task
	// I can continue wait with longer time
	rawResult, err := tsk.WaitWithTimeout(ctx, 2*time.Second)
	assert.NoError(t, err)
	assert.Equal(t, 9, rawResult)

	// any following Wait should complete immediately
	rawResult, err = tsk.WaitWithTimeout(ctx, 2*time.Nanosecond)
	assert.NoError(t, err)
	assert.Equal(t, 9, rawResult)
}

func TestPanicCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	tsk := asynctask.Start(ctx, getPanicTask(200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(ctx, 300*time.Millisecond)
	assert.True(t, errors.Is(err, asynctask.ErrPanic), "expecting ErrPanic")
}

func TestErrorCase(t *testing.T) {
	t.Parallel()
	ctx, cancelFunc := newTestContextWithTimeout(t, 3*time.Second)
	defer cancelFunc()

	tsk := asynctask.Start(ctx, getErrorTask("dummy error", 200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(ctx, 300*time.Millisecond)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, asynctask.ErrPanic), "not expecting ErrPanic")
	assert.False(t, errors.Is(err, context.DeadlineExceeded), "not expecting DeadlineExceeded")
	assert.Equal(t, "dummy error", err.Error())
}
