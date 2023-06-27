package asynctask

import (
	"context"
	"fmt"
)

// WaitAnyOptions defines options for WaitAny function
type WaitAnyOptions struct {
	// FailOnAnyError set to true will indicate WaitAny to return on first error it sees.
	FailOnAnyError bool
}

// WaitAny block current thread til any of task finished.
// first error from any tasks passed in will be returned if FailOnAnyError is set.
// first task end without error will end wait and return nil
func WaitAny(ctx context.Context, options *WaitAnyOptions, tasks ...Waitable) error {
	tasksCount := len(tasks)
	if tasksCount == 0 {
		return nil
	}

	if options == nil {
		options = &WaitAnyOptions{}
	}

	// tried to close channel before exit this func,
	// but it's complicated with routines, and we don't want to delay the return.
	// per https://stackoverflow.com/questions/8593645/is-it-ok-to-leave-a-channel-open, its ok to leave channel open, eventually it will be garbage collected.
	// this assumes the tasks eventually finish, otherwise we will have a routine leak.
	errorCh := make(chan error, tasksCount)

	for _, tsk := range tasks {
		go waitOne(ctx, tsk, errorCh)
	}

	runningTasks := tasksCount
	var errList []error
	for {
		select {
		case err := <-errorCh:
			runningTasks--
			if err != nil {
				// return immediately after receive first error if FailOnAnyError is set.
				if options.FailOnAnyError {
					return err
				}
				errList = append(errList, err)
			} else {
				// return immediately after first task completed.
				return nil
			}
		case <-ctx.Done():
			return fmt.Errorf("WaitAny %w", ctx.Err())
		}

		// are we finished yet?
		if runningTasks == 0 {
			break
		}
	}

	// when all tasks failed and FailOnAnyError is not set, return first one.
	// caller can get error for individual task by using Wait(),
	// it would return immediately after this WaitAny()
	return errList[0]
}
