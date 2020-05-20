package asynctask

import "context"

// ContinueFunc is a function that can be connected to previous task with ContinueWith
type ContinueFunc func(context.Context, interface{}) (interface{}, error)

// ContinueWith start the function when current task is done.
// result from previous task will be passed in, if no error.
func (tsk *TaskStatus) ContinueWith(ctx context.Context, next ContinueFunc) *TaskStatus {
	return Start(ctx, func(fCtx context.Context) (interface{}, error) {
		result, err := tsk.Wait(fCtx)
		if err != nil {
			return nil, err
		}
		return next(fCtx, result)
	})
}
