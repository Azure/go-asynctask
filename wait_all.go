package asynctask

import (
	"context"
	"fmt"
)

type Waitable interface {
	Wait(context.Context) error
}

// WaitAllOptions defines options for WaitAll function
type WaitAllOptions struct {
	// FailFast set to true will indicate WaitAll to return on first error it sees.
	FailFast bool
}

// WaitAll block current thread til all task finished.
// first error from any tasks passed in will be returned.
func WaitAll(ctx context.Context, options *WaitAllOptions, tasks ...Waitable) error {
	tasksCount := len(tasks)
	if tasksCount == 0 {
		return nil
	}

	if options == nil {
		options = &WaitAllOptions{}
	}

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
				// return immediately after receive first error.
				if options.FailFast {
					return err
				}
				errList = append(errList, err)
			}
		case <-ctx.Done():
			return fmt.Errorf("WaitAll %w", ctx.Err())
		}

		// are we finished yet?
		if runningTasks == 0 {
			break
		}
	}

	// we have at least 1 error, return first one.
	// caller can get error for individual task by using Wait(),
	// it would return immediately after this WaitAll()
	if len(errList) > 0 {
		return errList[0]
	}

	// no error at all.
	return nil
}

func waitOne(ctx context.Context, tsk Waitable, errorCh chan<- error) {
	err := tsk.Wait(ctx)
	errorCh <- err
}
