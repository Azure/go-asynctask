package asynctask

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"
)

// AsyncFunc is a function interface this asyncTask accepts.
type AsyncFunc[T any] func(context.Context) (T, error)

// ActionToFunc converts an Action to a Func (C# term), satisfying the AsyncFunc interface.
//
// - An Action is a function that performs an operation without returning a value.
// - A Func is a function that performs an operation and returns a value.
//
// The returned Func returns nil as the result and the original
// Action's error as the error value.
func ActionToFunc(action func(context.Context) error) func(context.Context) (any, error) {
	return func(ctx context.Context) (any, error) {
		return nil, action(ctx)
	}
}

// Task represents a handle to a running function.
// It provides methods to wait for completion, cancel execution, and retrieve results or errors.
type Task[T any] struct {
	state      State
	result     T
	err        error
	cancelFunc context.CancelFunc
	waitGroup  *sync.WaitGroup
	mutex      *sync.RWMutex
}

// State returns the current state of the task.
func (t *Task[T]) State() State {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.state
}

// Cancel cancels the task, by canceling the context.
// !! it relies on the task function to properly handle context cancellation.
// If the task has already finished, this method returns false.
func (t *Task[T]) Cancel() bool {
	if !t.finished() {
		t.finish(StateCanceled, *new(T), ErrCanceled)
		return true
	}

	return false
}

// Wait block current thread/routine until task finished or failed.
// context passed in can terminate the wait, through context cancellation
// but won't terminate the task (unless it's same context)
func (t *Task[T]) Wait(ctx context.Context) error {
	// return immediately if task already in terminal state.
	if t.finished() {
		return t.err
	}

	ch := make(chan any)
	go func() {
		t.waitGroup.Wait()
		close(ch)
	}()

	select {
	case <-ch:
		return t.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// WaitWithTimeout block current thread/routine until task finished or failed, or exceed the duration specified.
// timeout only stop waiting, taks will remain running.
func (t *Task[T]) WaitWithTimeout(ctx context.Context, timeout time.Duration) (T, error) {
	// return immediately if task already in terminal state.
	if t.finished() {
		return t.result, t.err
	}

	ctx, cancelFunc := context.WithTimeout(ctx, timeout)
	defer cancelFunc()

	return t.Result(ctx)
}

func (t *Task[T]) Result(ctx context.Context) (T, error) {
	err := t.Wait(ctx)
	if err != nil {
		return *new(T), err
	}

	return t.result, t.err
}


// Start starts an asynchronous task and returns a Task handle to manage it.
// The provided context can be used to cancel the task.
// You can use the returned Task to Wait for completion or Cancel the task.
func Start[T any](ctx context.Context, task AsyncFunc[T]) *Task[T] {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	mutex := &sync.RWMutex{}

	record := &Task[T]{
		state:      StateRunning,
		result:     *new(T),
		cancelFunc: cancel,
		waitGroup:  wg,
		mutex:      mutex,
	}

	go runAndTrackGenericTask(ctx, record, task)

	return record
}

// NewCompletedTask returns a Completed task, with result=nil, error=nil
func NewCompletedTask[T any](value T) *Task[T] {
	return &Task[T]{
		state:  StateCompleted,
		result: value,
		err:    nil,
		// nil cancelFunc and waitGroup should be protected with IsTerminalState()
		cancelFunc: nil,
		waitGroup:  nil,
		mutex:      &sync.RWMutex{},
	}
}

// runAndTrackGenericTask runs the given task and updates the provided Task record.
// It handles panics, errors, and task completion.
func runAndTrackGenericTask[T any](ctx context.Context, record *Task[T], task func(ctx context.Context) (T, error)) {
	defer record.waitGroup.Done()

	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic cought: %v, stackTrace: %s, %w", r, debug.Stack(), ErrPanic)
			record.finish(StateFailed, *new(T), err)
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

// finish updates the task's state, result, and error if it hasn't finished yet.
// It also cancels the underlying context.
func (t *Task[T]) finish(state State, result T, err error) {
	// only update state and result if not yet canceled
	t.mutex.Lock()
	defer t.mutex.Unlock()
	if !t.state.IsTerminalState() {
		t.cancelFunc() // cancel the context
		t.state = state
		t.result = result
		t.err = err
	}
}

// finished returns true if the task has reached a terminal state.
func (t *Task[T]) finished() bool {
	t.mutex.RLock()
	defer t.mutex.RUnlock()
	return t.state.IsTerminalState()
}
