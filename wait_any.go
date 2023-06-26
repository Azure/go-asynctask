package asynctask

import (
	"context"
	"fmt"
)

// WaitAny block current thread til any of task finished.
// first error from any tasks passed in will be returned
// first task end without error will end wait and return nil
func WaitAny(ctx context.Context, tasks ...Waitable) error {
	tasksCount := len(tasks)
	if tasksCount == 0 {
		return nil
	}

	// tried to close channel before exit this func,
	// but it's complicated with routines, and we don't want to delay the return.
	// per https://stackoverflow.com/questions/8593645/is-it-ok-to-leave-a-channel-open, its ok to leave channel open, eventually it will be garbage collected.
	// this assumes the tasks eventually finish, otherwise we will have a routine leak.
	errorCh := make(chan error, tasksCount)

	for _, tsk := range tasks {
		go waitOne(ctx, tsk, errorCh)
	}

	for {
		select {
		case err := <-errorCh:
			return err
		case <-ctx.Done():
			return fmt.Errorf("WaitAny %w", ctx.Err())
		}
	}
}
