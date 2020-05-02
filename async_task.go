package asynctask

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"sync"
	"time"
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

// IsTerminalState tells whether the task finished
func (s State) IsTerminalState() bool {
	return s != StateRunning
}

// AsyncFunc is a function interface this asyncTask accepts.
type AsyncFunc func(context.Context) (interface{}, error)

// ErrPanic is returned if panic cought in the task
var ErrPanic = errors.New("panic")

// ErrTimeout is returned if task didn't finish within specified time duration.
var ErrTimeout = errors.New("timeout")

// ErrWaitTimeout is returned on Wait or WaitWithTimeout.
var ErrWaitTimeout = errors.New("wait timeout")

// ErrCanceled is returned if a cancel is triggered
var ErrCanceled = errors.New("canceled")

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
	if !t.state.IsTerminalState() {
		t.cancelFunc()

		t.finish(StateCanceled, nil, ErrCanceled)
	}
}

// Wait block current thread/routine until task finished or failed.
// context passed in can terminate the wait, through context cancellation
// but won't terminate the task (unless it's same context)
func (t *TaskStatus) Wait(ctx context.Context) (interface{}, error) {
	// return immediately if task already in terminal state.
	if t.state.IsTerminalState() {
		return t.result, t.err
	}

	ch := make(chan interface{})
	go func() {
		t.waitGroup.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		return t.result, t.err
	case <-ctx.Done():
		return nil, ErrWaitTimeout
	}
}

// WaitWithTimeout block current thread/routine until task finished or failed, or exceed the duration specified.
// timeout only stop waiting, taks will remain running.
func (t *TaskStatus) WaitWithTimeout(ctx context.Context, timeout time.Duration) (interface{}, error) {
	// return immediately if task already in terminal state.
	if t.state.IsTerminalState() {
		return t.result, t.err
	}

	ctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()

	return t.Wait(ctx)
}

// NewCompletedTask returns a Completed task, with result=nil, error=nil
func NewCompletedTask() *TaskStatus {
	return &TaskStatus{
		state:  StateCompleted,
		result: nil,
		err:    nil,
		// nil cancelFunc and waitGroup should be protected with IsTerminalState()
		cancelFunc: nil,
		waitGroup:  nil,
	}
}

// Start run a async function and returns you a handle which you can Wait or Cancel.
// context passed in may impact task lifetime (from context cancellation)
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

// isErrorReallyError do extra error check
//    - Nil Pointer to a Type (that implement error)
//    - Zero Value of a Type (that implement error)
func isErrorReallyError(err error) bool {
	v := reflect.ValueOf(err)
	if v.Type().Kind() == reflect.Ptr &&
		v.IsNil() {
		return false
	}

	if v.Type().Kind() == reflect.Struct &&
		v.IsZero() {
		return false
	}
	return true
}

func runAndTrackTask(record *TaskStatus, task func(ctx context.Context) (interface{}, error)) {
	defer record.waitGroup.Done()
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("Panic cought: %v, StackTrace: %s, %w", r, debug.Stack(), ErrPanic)
			record.finish(StateFailed, nil, err)
		}
	}()

	result, err := task(record)

	if err == nil ||
		// incase some team use pointer typed error (implement Error() string on a pointer type)
		// which can break err check (but nil point assigned to error result to non-nil error)
		// check out TestPointerErrorCase in error_test.go
		!isErrorReallyError(err) {
		record.finish(StateCompleted, result, nil)
		return
	}

	// err not nil, fail the task
	record.finish(StateFailed, result, err)
}

func (t *TaskStatus) finish(state State, result interface{}, err error) {
	// only update state and result if not yet canceled
	if !t.state.IsTerminalState() {
		t.state = state
		t.result = result
		t.err = err
	}
}
