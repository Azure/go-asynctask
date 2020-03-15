package asynctask_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func getPanicTask(sleepDuration time.Duration) asynctask.AsyncFunc {
	return func(ctx context.Context) (interface{}, error) {
		time.Sleep(sleepDuration)
		panic("yo")
	}
}

func getErrorTask(sleepDuration time.Duration) asynctask.AsyncFunc {
	return func(ctx context.Context) (interface{}, error) {
		time.Sleep(sleepDuration)
		return nil, errors.New("not found")
	}
}

func TestTimeoutCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	tsk := asynctask.Start(ctx, getCountingTask(200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(300 * time.Millisecond)
	assert.True(t, errors.Is(err, asynctask.ErrTimeout), "expecting ErrTimeout")
}

func TestPanicCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	tsk := asynctask.Start(ctx, getPanicTask(200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(300 * time.Millisecond)
	assert.True(t, errors.Is(err, asynctask.ErrPanic), "expecting ErrPanic")
}

func TestErrorCase(t *testing.T) {
	t.Parallel()
	ctx := newTestContext(t)
	tsk := asynctask.Start(ctx, getErrorTask(200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(300 * time.Millisecond)
	assert.Error(t, err)
	assert.False(t, errors.Is(err, asynctask.ErrPanic), "not expecting ErrPanic")
	assert.False(t, errors.Is(err, asynctask.ErrTimeout), "not expecting ErrTimeout")
}
