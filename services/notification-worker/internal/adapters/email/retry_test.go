package email

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestRetryPolicyExecuteRetriesTransientFailure(t *testing.T) {
	policy := testRetryPolicy(t, 3)
	attempts := 0
	var delays []time.Duration

	err := policy.execute(
		context.Background(),
		func() error {
			attempts++
			if attempts < 3 {
				return errors.New("temporary failure")
			}
			return nil
		},
		func(error) bool { return true },
		func(_ int, delay time.Duration, _ error) { delays = append(delays, delay) },
	)
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if attempts != 3 {
		t.Errorf("got %d attempts, want 3", attempts)
	}
	wantDelays := []time.Duration{time.Millisecond, 2 * time.Millisecond}
	for i := range wantDelays {
		if delays[i] != wantDelays[i] {
			t.Errorf("got delay %s at index %d, want %s", delays[i], i, wantDelays[i])
		}
	}
}

func TestRetryPolicyExecuteDoesNotRetryPermanentFailure(t *testing.T) {
	policy := testRetryPolicy(t, 3)
	wantErr := errors.New("permanent failure")
	attempts := 0

	err := policy.execute(
		context.Background(),
		func() error {
			attempts++
			return wantErr
		},
		func(error) bool { return false },
		func(int, time.Duration, error) {},
	)

	if !errors.Is(err, wantErr) {
		t.Fatalf("got error %v, want permanent failure", err)
	}
	if attempts != 1 {
		t.Errorf("got %d attempts, want 1", attempts)
	}
}

func TestRetryPolicyExecuteReturnsExhaustion(t *testing.T) {
	policy := testRetryPolicy(t, 3)
	attempts := 0

	err := policy.execute(
		context.Background(),
		func() error {
			attempts++
			return errors.New("temporary failure")
		},
		func(error) bool { return true },
		func(int, time.Duration, error) {},
	)

	if !errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("got error %v, want retry exhaustion", err)
	}
	if attempts != 3 {
		t.Errorf("got %d attempts, want 3", attempts)
	}
}

func TestRetryPolicyExecuteStopsWhenContextIsCanceled(t *testing.T) {
	policy := testRetryPolicy(t, 3)
	ctx, cancel := context.WithCancel(context.Background())
	policy.waitForBackoff = func(context.Context, time.Duration) error {
		cancel()
		return ctx.Err()
	}

	err := policy.execute(
		ctx,
		func() error { return errors.New("temporary failure") },
		func(error) bool { return true },
		func(int, time.Duration, error) {},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got error %v, want context cancellation", err)
	}
}

func TestRetryPolicyDelayAfterUsesExponentialBackoffWithCap(t *testing.T) {
	policy, err := NewRetryPolicy(6, 500*time.Millisecond, 5*time.Second)
	if err != nil {
		t.Fatalf("create retry policy: %v", err)
	}

	tests := []struct {
		afterAttempt int
		want         time.Duration
	}{
		{afterAttempt: 1, want: 500 * time.Millisecond},
		{afterAttempt: 2, want: time.Second},
		{afterAttempt: 3, want: 2 * time.Second},
		{afterAttempt: 4, want: 4 * time.Second},
		{afterAttempt: 5, want: 5 * time.Second},
	}

	for _, tt := range tests {
		if got := policy.delayAfter(tt.afterAttempt); got != tt.want {
			t.Errorf("delay after attempt %d = %s, want %s", tt.afterAttempt, got, tt.want)
		}
	}
}

func TestNewRetryPolicyValidatesConfiguration(t *testing.T) {
	tests := []struct {
		name         string
		maxAttempts  int
		initialDelay time.Duration
		maxDelay     time.Duration
	}{
		{name: "zero attempts", maxAttempts: 0, initialDelay: time.Second, maxDelay: time.Second},
		{name: "zero initial delay", maxAttempts: 1, initialDelay: 0, maxDelay: time.Second},
		{name: "max below initial", maxAttempts: 1, initialDelay: time.Second, maxDelay: time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewRetryPolicy(tt.maxAttempts, tt.initialDelay, tt.maxDelay); err == nil {
				t.Fatal("got nil error, want invalid retry policy error")
			}
		})
	}
}
