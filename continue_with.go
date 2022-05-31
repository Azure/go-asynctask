package asynctask

import "context"

// ContinueFunc is a function that can be connected to previous task with ContinueWith
type ContinueFunc[T any, S any] func(context.Context, *T) (*S, error)

func ContinueWith[T any, S any](ctx context.Context, tsk *Task[T], next ContinueFunc[T, S]) *Task[S] {
	return Start(ctx, func(fCtx context.Context) (*S, error) {
		result, err := tsk.Result(fCtx)
		if err != nil {
			return nil, err
		}
		return next(fCtx, result)
	})
}
