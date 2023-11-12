package asynctask

import "context"

// AfterBothFunc is a function that has 2 input.
type AfterBothFunc[T, S, R any] func(context.Context, T, S) (R, error)

// AfterBoth runs the function after both 2 input task finished, and will be fed with result from 2 input task.
//
//	if one of the input task failed, the AfterBoth task will be failed and returned, even other one are still running.
func AfterBoth[T, S, R any](ctx context.Context, tskT *Task[T], tskS *Task[S], next AfterBothFunc[T, S, R]) *Task[R] {
	return Start(ctx, func(fCtx context.Context) (R, error) {
		t, err := tskT.Result(fCtx)
		if err != nil {
			return *new(R), err
		}

		s, err := tskS.Result(fCtx)
		if err != nil {
			return *new(R), err
		}

		return next(fCtx, t, s)
	})
}

// AfterBothActionToFunc convert a Action to Func (C# term), to satisfy the AfterBothFunc interface.
//
//	Action is function that runs without return anything
//	Func is function that runs and return something
func AfterBothActionToFunc[T, S any](action func(context.Context, T, S) error) func(context.Context, T, S) (interface{}, error) {
	return func(ctx context.Context, t T, s S) (interface{}, error) {
		return nil, action(ctx, t, s)
	}
}
