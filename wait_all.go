package asynctask

import (
	"context"
	"fmt"
	"sync"
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

	mutex := sync.Mutex{}
	errorChClosed := false
	errorCh := make(chan error, tasksCount)
	// hard close channel
	defer close(errorCh)

	for _, tsk := range tasks {
		go waitOne(ctx, tsk, errorCh, &errorChClosed, &mutex)
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
					softCloseChannel(&mutex, &errorChClosed)
					return err
				}

				errList = append(errList, err)
			}
		case <-ctx.Done():
			softCloseChannel(&mutex, &errorChClosed)
			return fmt.Errorf("WaitAll context canceled: %w", ctx.Err())
		}

		// are we finished yet?
		if runningTasks == 0 {
			softCloseChannel(&mutex, &errorChClosed)
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

func waitOne(ctx context.Context, tsk Waitable, errorCh chan<- error, errorChClosed *bool, mutex *sync.Mutex) {
	err := tsk.Wait(ctx)

	// why mutex?
	// if all tasks start using same context (unittest is good example)
	// and that context got canceled, all task fail at same time.
	// first one went in and close the channel, while another one already went through gate check.
	// raise a panic with send to closed channel.
	mutex.Lock()
	defer mutex.Unlock()
	if !*errorChClosed {
		errorCh <- err
	}
}

func softCloseChannel(mutex *sync.Mutex, closed *bool) {
	mutex.Lock()
	defer mutex.Unlock()
	*closed = true
}
