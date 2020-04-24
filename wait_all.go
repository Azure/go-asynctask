package asynctask

import (
	"context"
	"fmt"
)

// WaitAllOptions defines options for WaitAll function
type WaitAllOptions struct {
	// FailFast set to true will indicate WaitAll to return on first error it sees.
	FailFast bool
}

// WaitAll block current thread til all task finished.
// it return immediately after receive first error.
func WaitAll(ctx context.Context, options *WaitAllOptions, tasks ...*TaskStatus) error {
	errorCh := make(chan error)
	errorChClosed := false
	defer func() {
		errorChClosed = true
		close(errorCh)
	}()

	runningTasks := 0
	for _, tsk := range tasks {

		go func(tsk *TaskStatus) {
			_, err := tsk.Wait()
			if !errorChClosed {
				errorCh <- err
			}
		}(tsk)
		runningTasks++
	}

	var errList []error
	for {
		select {
		case err := <-errorCh:
			runningTasks--
			if err != nil && isErrorReallyError(err) {
				// return immediately after receive first error.
				if options.FailFast {
					return err
				}

				errList = append(errList, err)
			}
		case <-ctx.Done():
			return fmt.Errorf("WaitAll context canceled. %w", context.Canceled)
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
