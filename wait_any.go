package asynctask

import (
	"context"
	"fmt"
)

// WaitAnyOptions defines options for WaitAny function
type WaitAnyOptions struct {
	// FailFast set to true will indicate WaitAny to return on first error it sees.
	FailFast bool
}

// WaitAny block current thread til any of task finished.
// first error from any tasks passed in will be returned if FailFast is set.
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
				// return immediately after receive first error if FailFast is set.
				if options.FailFast {
					return err
				}
				errList = append(errList, err)
			} else {
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

	// we have at least 1 error when FailFast is not set, return first one.
	// caller can get error for individual task by using Wait(),
	// it would return immediately after this WaitAny()
	if len(errList) > 0 {
		return errList[0]
	}

	// no error at all.
	return nil
}
