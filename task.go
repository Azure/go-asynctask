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

// ActionToFunc convert a Action to Func (C# term), to satisfy the AsyncFunc interface.
// -  Action is function that runs without return anything
// -  Func is function that runs and return something
func ActionToFunc(action func(context.Context) error) func(context.Context) (*interface{}, error) {
	return func(ctx context.Context) (*interface{}, error) {
		return nil, action(ctx)
	}
}

// Task is a handle to the running function.
// which you can use to wait, cancel, get the result.
type Task[T any] struct {
	state      State
	result     T
	err        error
	cancelFunc context.CancelFunc
	waitGroup  *sync.WaitGroup
	mutex      *sync.Mutex
}

// State return state of the task.
func (t *Task[T]) State() State {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.state
}

// Cancel the task by cancel the context.
// !! this rely on the task function to check context cancellation and proper context handling.
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

	ch := make(chan interface{})
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

// Start run a async function and returns you a handle which you can Wait or Cancel.
// context passed in may impact task lifetime (from context cancellation)
func Start[T any](ctx context.Context, task AsyncFunc[T]) *Task[T] {
	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	wg.Add(1)
	mutex := &sync.Mutex{}

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
		mutex:      &sync.Mutex{},
	}
}

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

func (t *Task[T]) finished() bool {
	t.mutex.Lock()
	defer t.mutex.Unlock()
	return t.state.IsTerminalState()
}
