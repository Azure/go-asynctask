package taskStore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/haitch/taskStore"
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
	fmt.Println("Task Queued")
	result := ts.GetTask("t1")
	fmt.Println(result)
	time.Sleep(time.Second * 3)
	result = ts.GetTask("t1")
	fmt.Println(result)
}
