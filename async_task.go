package asynctask

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"
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

// ErrCanceled is returned if a cancel is triggered
var ErrCanceled = errors.New("canceled")

// TaskStatus is a handle to the running function.
// which you can use to wait, cancel, get the result.
type TaskStatus struct {
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
		return nil, ctx.Err()
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
		state:      StateRunning,
		result:     nil,
		cancelFunc: cancel,
		waitGroup:  wg,
	}

	go runAndTrackTask(ctx, record, task)

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

func identifypanic() (runtime.Frame, string) {

	panicFrame := runtime.Frame{
		Function: "UNKNOWN",
		File:     "UNKNOWN",
	}
	pc := make([]uintptr, 16)               //how deep should we look? 16 a deep enough stack?
	n := runtime.Callers(2, pc)             // 3 == runtime.Callers, IdentifyPanic.
	frames := runtime.CallersFrames(pc[:n]) // pass only valid pcs to runtime.CallersFrames

	stackhash := md5.New()
	foundPanic := false
	passedPanic := false
	for {
		frame, more := frames.Next()

		//ignore runtime at beginning and end of stack
		if strings.HasPrefix(frame.Function, "runtime.") {
			passedPanic = true
			continue
		}

		//could be other non runtime funcions before panic
		if !passedPanic {
			continue
		}

		if !foundPanic {
			//frame that raised the panic will be first after runtime
			//but the number of runtime lines can vary so can't just
			panicFrame = frame
			foundPanic = true
		}
		_, _ = stackhash.Write([]byte(fmt.Sprintf("%s%d\n", frame.Function, frame.Line)))

		if !more {
			break
		}
	}
	return panicFrame, fmt.Sprintf("%x", stackhash.Sum(nil))
}

type panicErr struct {
	frame     runtime.Frame
	stackhash string
	recovery  interface{}
}

func (pe *panicErr) Error() string {
	bytes, _ := json.Marshal(pe)
	return string(bytes)
}

func runAndTrackTask(ctx context.Context, record *TaskStatus, task func(ctx context.Context) (interface{}, error)) {
	defer record.waitGroup.Done()
	defer func() {
		if r := recover(); r != nil {
			frame, hash := identifypanic()
			err := &panicErr{
				frame:     frame,
				stackhash: hash,
				recovery:  r,
			}
			record.finish(StateFailed, nil, err)
		}
	}()

	result, err := task(ctx)

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
