package asynctask

import (
	"errors"
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

// ErrPanic is returned if panic cought in the task
var ErrPanic = errors.New("panic")

// ErrCanceled is returned if a cancel is triggered
var ErrCanceled = errors.New("canceled")
