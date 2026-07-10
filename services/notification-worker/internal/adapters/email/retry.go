package email

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrRetryExhausted = errors.New("retry attempts exhausted")

type RetryPolicy struct {
	maxAttempts    int
	initialDelay   time.Duration
	maxDelay       time.Duration
	waitForBackoff func(context.Context, time.Duration) error
}

func NewRetryPolicy(maxAttempts int, initialDelay, maxDelay time.Duration) (RetryPolicy, error) {
	if maxAttempts <= 0 {
		return RetryPolicy{}, fmt.Errorf("retry max attempts must be positive")
	}
	if initialDelay <= 0 {
		return RetryPolicy{}, fmt.Errorf("retry initial delay must be positive")
	}
	if maxDelay < initialDelay {
		return RetryPolicy{}, fmt.Errorf("retry max delay must be greater than or equal to initial delay")
	}

	return RetryPolicy{
		maxAttempts:    maxAttempts,
		initialDelay:   initialDelay,
		maxDelay:       maxDelay,
		waitForBackoff: waitForBackoff,
	}, nil
}

func (p RetryPolicy) execute(
	ctx context.Context,
	operation func() error,
	shouldRetry func(error) bool,
	onRetry func(attempt int, delay time.Duration, err error),
) error {
	for attempt := 1; attempt <= p.maxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		err := operation()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !shouldRetry(err) {
			return err
		}
		if attempt == p.maxAttempts {
			return fmt.Errorf("%w after %d attempts: %w", ErrRetryExhausted, attempt, err)
		}

		delay := p.delayAfter(attempt)
		onRetry(attempt, delay, err)
		if err := p.waitForBackoff(ctx, delay); err != nil {
			return err
		}
	}

	return nil
}

func (p RetryPolicy) delayAfter(attempt int) time.Duration {
	delay := p.initialDelay
	for retry := 1; retry < attempt; retry++ {
		if delay >= p.maxDelay/2 {
			return p.maxDelay
		}
		delay *= 2
	}
	if delay > p.maxDelay {
		return p.maxDelay
	}
	return delay
}

func waitForBackoff(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
