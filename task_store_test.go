package taskStore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/haitch/taskStore"
	"github.com/stretchr/testify/assert"
)

func countingTask(ctx context.Context) (interface{}, error) {
	result := 0
	for i := 0; i < 10; i++ {
		time.Sleep(time.Millisecond * 200)
		fmt.Printf("  working %d\n", i)
		result = i
	}
	return result, nil
}

func TestEasyCase(t *testing.T) {
	t.Parallel()

	ts := taskStore.New()
	ts.RegistTask("t1", countingTask)

	result := ts.GetTask("t1")
	assert.Equal(t, result.Status, taskStore.TaskStatusRunning, "Task should queued to Running")

	time.Sleep(time.Second * 3)

	result = ts.GetTask("t1")
	assert.Equal(t, result.Status, taskStore.TaskStatusCompleted, "Task should complete by now")
}
