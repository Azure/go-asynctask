package asynctask

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

// StateCanceled indicate task got canceled.
const StateCanceled State = "Canceled"

// AsyncFunc is a function interface this asyncTask accepts.
type AsyncFunc func(context.Context) (interface{}, error)

// TaskStatus is a handle to the running function.
// which you can use to wait, cancel, get the result.
type TaskStatus struct {
	context.Context
	state      State
	result     interface{}
	err        error
	cancelFunc context.CancelFunc
	waitGroup  *sync.WaitGroup
}

// State return state of the task.
func (t *TaskStatus) State() State {
	return t.state
}

// Cancel abort the task execution
// !! only if the function provided handles context cancel.
func (t *TaskStatus) Cancel() {
	t.cancelFunc()

	// could have a debate wait or not
	// my point is wait drives this TaskStatus to a terminal(consistence) state.
	//	 you can keep Cancel() or Wait() or State() on this handle, and nothing changes.
	// without this wait
	//   1. nothing blocks invoking Wait() on this handle after Cancel()
	//   2. the state, result, error, can be non-deterministic.
	t.waitGroup.Wait()

	t.state = StateCanceled
}

// Wait block current thread/routine until task finished or failed.
func (t *TaskStatus) Wait() (interface{}, error) {
	t.waitGroup.Wait()

	// we create new context when starting task, now release it.
	t.cancelFunc()

	return t.result, t.err
}

// Start run a async function and returns you a handle which you can Wait or Cancel.
func Start(ctx context.Context, task AsyncFunc) *TaskStatus {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	record := &TaskStatus{
		Context:    ctx,
		state:      StateRunning,
		result:     nil,
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
			record.state = StateFailed
			record.err = err
			record.waitGroup.Done()
		}
	}()

	result, err := task(record)
	if err != nil {
		record.state = StateFailed
		record.err = err
		record.waitGroup.Done()
	} else {
		record.state = StateCompleted
		record.result = result
		record.waitGroup.Done()
	}
}
