package asynctask

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// AsyncFunc is a function interface this asyncTask accepts.
type AsyncGenericFunc[T any] func(context.Context) (*T, error)

// Task is a handle to the running function.
// which you can use to wait, cancel, get the result.
type Task[T any] struct {
	state      State
	result     *T
	err        error
	cancelFunc context.CancelFunc
	waitGroup  *sync.WaitGroup
}

// State return state of the task.
func (t *Task[T]) State() State {
	return t.state
}

// Cancel abort the task execution
// !! only if the function provided handles context cancel.
func (t *Task[T]) Cancel() {
	if !t.state.IsTerminalState() {
		t.cancelFunc()

		var result T
		t.finish(StateCanceled, &result, ErrCanceled)
	}
}

// Wait block current thread/routine until task finished or failed.
// context passed in can terminate the wait, through context cancellation
// but won't terminate the task (unless it's same context)
func (t *Task[T]) Wait(ctx context.Context) (*T, error) {
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
		return nil, ctx.Err()
	}
}

// WaitWithTimeout block current thread/routine until task finished or failed, or exceed the duration specified.
// timeout only stop waiting, taks will remain running.
func (t *Task[T]) WaitWithTimeout(ctx context.Context, timeout time.Duration) (*T, error) {
	// return immediately if task already in terminal state.
	if t.state.IsTerminalState() {
		return t.result, t.err
	}

	ctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()

	return t.Wait(ctx)
}

// Start run a async function and returns you a handle which you can Wait or Cancel.
// context passed in may impact task lifetime (from context cancellation)
func StartGeneric[T any](ctx context.Context, task AsyncGenericFunc[T]) *Task[T] {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)

	var result T
	record := &Task[T]{
		state:      StateRunning,
		result:     &result,
		cancelFunc: cancel,
		waitGroup:  wg,
	}

	go runAndTrackGenericTask(ctx, record, task)

	return record
}

// NewCompletedTask returns a Completed task, with result=nil, error=nil
func NewCompletedGenericTask[T any](value *T) *Task[T] {
	return &Task[T]{
		state:  StateCompleted,
		result: value,
		err:    nil,
		// nil cancelFunc and waitGroup should be protected with IsTerminalState()
		cancelFunc: nil,
		waitGroup:  nil,
	}
}

func runAndTrackGenericTask[T any](ctx context.Context, record *Task[T], task func(ctx context.Context) (*T, error)) {
	defer record.waitGroup.Done()
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("Panic cought: %v, StackTrace: %s, %w", r, debug.Stack(), ErrPanic)
			record.finish(StateFailed, nil, err)
		}
	}()

	result, err := task(ctx)

	if err == nil {
		record.finish(StateCompleted, result, nil)
		return
	}

	// err not nil, fail the task
	record.finish(StateFailed, result, err)
}

func (t *Task[T]) finish(state State, result *T, err error) {
	// only update state and result if not yet canceled
	if !t.state.IsTerminalState() {
		t.state = state
		t.result = result
		t.err = err
	}
}
