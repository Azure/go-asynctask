package taskstore

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
)

// State of a task.
type State string

// StateRunning indicate task is still running.
const StateRunning State = "Running"

// StateCompleted indicate task is finished.
const StateCompleted State = "Completed"

// StateFailed indicate task failed.
const StateFailed State = "Failed"

// TaskStatus is a handle to the
type TaskStatus struct {
	context.Context
	State      State
	Result     interface{}
	Error      error
	cancelFunc context.CancelFunc
	waitGroup  *sync.WaitGroup
}

// Cancel abort the task execution
// !! only if the function provided handles context cancel.
func (t *TaskStatus) Cancel() {
	t.cancelFunc()
	t.waitGroup.Wait()
}

// Wait block current thread/routine until task finished or failed.
func (t *TaskStatus) Wait() {
	t.waitGroup.Wait()
}

// StartTask returns you a handle which you can Wait or Cancel.
func StartTask(ctx context.Context, task func(ctx context.Context) (interface{}, error)) *TaskStatus {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	record := &TaskStatus{
		Context:    ctx,
		State:      StateRunning,
		Result:     nil,
		cancelFunc: cancel,
		waitGroup:  wg,
	}

	go runAndTrackTask(record, task)

	return record
}

func runAndTrackTask(record *TaskStatus, task func(ctx context.Context) (interface{}, error)) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("Panic cought: %v, StackTrace: %s", r, debug.Stack())
			record.State = StateFailed
			record.Error = err
			record.waitGroup.Done()
		}
	}()

	result, err := task(record)
	if err != nil {
		record.State = StateFailed
		record.Error = err
		record.waitGroup.Done()
	} else {
		record.State = StateCompleted
		record.Result = result
		record.waitGroup.Done()
	}
}
