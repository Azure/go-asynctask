package asynctask

import "context"

// AfterBothFunc is a function that has 2 input.
type AfterBothFunc[T, S, R any] func(context.Context, *T, *S) (*R, error)

// AfterBoth runs the function after both 2 input task finished, and will be feed with result from 2 input task.
func AfterBoth[T, S, R any](ctx context.Context, tskT *Task[T], tskS *Task[S], next AfterBothFunc[T, S, R]) *Task[R] {
	return Start(ctx, func(fCtx context.Context) (*R, error) {
		t, err := tskT.Result(fCtx)
		if err != nil {
			return nil, err
		}

		s, err := tskS.Result(fCtx)
		if err != nil {
			return nil, err
		}

		return next(fCtx, t, s)
	})
}
