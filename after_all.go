package asynctask

import "context"

// AfterAll3Func is a function that has 3 input.
type AfterAll3Func[T, S, R, Q any] func(context.Context, *T, *S, *R) (*Q, error)

// AfterAll4Func is a function that has 4 input.
type AfterAll4Func[T, S, R, Q, P any] func(context.Context, *T, *S, *R, *Q) (*P, error)

// AfterAll5Func is a function that has 5 input.
type AfterAll5Func[T, S, R, Q, P, O any] func(context.Context, *T, *S, *R, *Q, *P) (*O, error)

// AfterAll3 runs the function after both 3 input task finished, and will be feed with result from 3 input task.
func AfterAll3[T, S, R, Q any](ctx context.Context, tskT *Task[T], tskS *Task[S], tskR *Task[R], next AfterAll3Func[T, S, R, Q]) *Task[Q] {
	return Start(ctx, func(fCtx context.Context) (*Q, error) {
		t, err := tskT.Result(fCtx)
		if err != nil {
			return nil, err
		}

		s, err := tskS.Result(fCtx)
		if err != nil {
			return nil, err
		}

		r, err := tskR.Result(fCtx)
		if err != nil {
			return nil, err
		}

		return next(fCtx, t, s, r)
	})
}

// AfterAll4 runs the function after both 4 input task finished, and will be feed with result from 4 input task.
func AfterAll4[T, S, R, Q, P any](ctx context.Context, tskT *Task[T], tskS *Task[S], tskR *Task[R], tskQ *Task[Q], next AfterAll4Func[T, S, R, Q, P]) *Task[P] {
	return Start(ctx, func(fCtx context.Context) (*P, error) {
		t, err := tskT.Result(fCtx)
		if err != nil {
			return nil, err
		}

		s, err := tskS.Result(fCtx)
		if err != nil {
			return nil, err
		}

		r, err := tskR.Result(fCtx)
		if err != nil {
			return nil, err
		}

		q, err := tskQ.Result(fCtx)
		if err != nil {
			return nil, err
		}

		return next(fCtx, t, s, r, q)
	})
}

// AfterAll5 runs the function after both 5 input task finished, and will be feed with result from 5 input task.
func AfterAll5[T, S, R, Q, P, O any](ctx context.Context, tskT *Task[T], tskS *Task[S], tskR *Task[R], tskQ *Task[Q], tskP *Task[P], next AfterAll5Func[T, S, R, Q, P, O]) *Task[O] {
	return Start(ctx, func(fCtx context.Context) (*O, error) {
		t, err := tskT.Result(fCtx)
		if err != nil {
			return nil, err
		}

		s, err := tskS.Result(fCtx)
		if err != nil {
			return nil, err
		}

		r, err := tskR.Result(fCtx)
		if err != nil {
			return nil, err
		}

		q, err := tskQ.Result(fCtx)
		if err != nil {
			return nil, err
		}

		p, err := tskP.Result(fCtx)
		if err != nil {
			return nil, err
		}

		return next(fCtx, t, s, r, q, p)
	})
}
