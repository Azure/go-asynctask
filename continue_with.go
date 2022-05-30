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

// ContinueFunc is a function that can be connected to previous task with ContinueWith
type ContinueFuncGeneric[T any, S any] func(context.Context, *T) (*S, error)

func ContinueWith[T any, S any](ctx context.Context, tsk *Task[T], next ContinueFuncGeneric[T, S]) *Task[S] {
	return StartGeneric(ctx, func(fCtx context.Context) (*S, error) {
		result, err := tsk.Wait(fCtx)
		if err != nil {
			return nil, err
		}
		return next(fCtx, result)
	})
}
