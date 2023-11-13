package asynctask

import "context"

// ContinueFunc is a function that can be connected to previous task with ContinueWith
type ContinueFunc[T any, S any] func(context.Context, T) (S, error)

func ContinueWith[T any, S any](ctx context.Context, tsk *Task[T], next ContinueFunc[T, S]) *Task[S] {
	return Start(ctx, func(fCtx context.Context) (S, error) {
		result, err := tsk.Result(fCtx)
		if err != nil {
			return *new(S), err
		}
		return next(fCtx, result)
	})
}

// ContinueActionToFunc convert a Action to Func (C# term), to satisfy the AsyncFunc interface.
//
//	Action is function that runs without return anything
//	Func is function that runs and return something
func ContinueActionToFunc[T any](action func(context.Context, T) error) func(context.Context, T) (interface{}, error) {
	return func(ctx context.Context, t T) (interface{}, error) {
		return nil, action(ctx, t)
	}
}
