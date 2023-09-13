package poll

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func Do(ctx context.Context, interval time.Duration, doFunc func(context.Context) error) error {
	df := func(ictx context.Context) (struct{}, error) {
		return struct{}{}, doFunc(ictx)
	}

	_, err := DoR(ctx, interval, df)
	return err
}

func DoR[R any](ctx context.Context, interval time.Duration, doFunc func(context.Context) (R, error)) (R, error) {
	// first attempt
	errs := []error{}
	if t, err := doFunc(ctx); err == nil {
		return t, err
	} else {
		errs = append(errs, err)
	}

	// loop until timeout
	tr := time.NewTimer(interval)
	for {
		select {
		case <-ctx.Done():
			var t R
			return t, fmt.Errorf("poller timed out: %w", errors.Join(errs...))
		case <-tr.C:
			if t, err := doFunc(ctx); err == nil {
				return t, nil
			} else {
				errs = append(errs, err)
			}

			tr.Reset(interval)
		}
	}
}
