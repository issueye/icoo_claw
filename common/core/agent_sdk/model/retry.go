package model

import (
	"context"
	"errors"
	"time"
)

const defaultRetryBaseDelay = 100 * time.Millisecond

type retryableFunc func(error) bool

func runWithRetry(ctx context.Context, maxRetries int, isRetryable retryableFunc, fn func(context.Context) error) error {
	if fn == nil {
		return errors.New("model: retry function is nil")
	}
	attempts := 0
	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if isRetryable == nil || !isRetryable(err) || attempts >= maxRetries {
			return err
		}
		attempts++
		backoff := retryBackoff(attempts)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
	}
}

func retryBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		attempt = 1
	}
	return time.Duration(attempt*attempt) * defaultRetryBaseDelay
}
