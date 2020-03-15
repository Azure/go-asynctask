package asynctask_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Azure/go-asynctask"
	"github.com/stretchr/testify/assert"
)

func TestTimeoutCase(t *testing.T) {
	ctx := newTestContext(t)
	tsk := asynctask.Start(ctx, getCountingTask(200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(300 * time.Millisecond)
	assert.True(t, errors.Is(err, asynctask.ErrTimeout), "expecting ErrTimeout")
}

func TestWeirdCase(t *testing.T) {
	ctx := newTestContext(t)
	tsk := asynctask.Start(ctx, getCountingTask(-200*time.Millisecond))
	_, err := tsk.WaitWithTimeout(300 * time.Millisecond)
	assert.True(t, errors.Is(err, asynctask.ErrTimeout), "expecting ErrTimeout")
}
