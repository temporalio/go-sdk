// The MIT License
//
// Copyright (c) 2020 Temporal Technologies Inc.  All rights reserved.
//
// Copyright (c) 2020 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package backoff

import (
	"context"
	"errors"
	"sync"
	"time"

	"go.temporal.io/api/serviceerror"
)

type (
	// Operation to retry
	Operation func() error

	// IsRetryable handler can be used to exclude certain errors during retry
	IsRetryable func(error) bool

	// ConcurrentRetrier is used for client-side throttling. It determines whether to
	// throttle outgoing traffic in case downstream backend server rejects
	// requests due to out-of-quota or server busy errors.
	ConcurrentRetrier struct {
		sync.Mutex
		retrier      Retrier // Backoff retrier
		failureCount int64   // Number of consecutive failures seen
	}
)

// Throttle Sleep if there were failures since the last success call.
func (c *ConcurrentRetrier) Throttle() {
	c.throttleInternal()
}

func (c *ConcurrentRetrier) throttleInternal() time.Duration {
	next := done

	// Check if we have failure count.
	c.Lock()
	if c.failureCount > 0 {
		next = c.retrier.NextBackOff()
	}
	c.Unlock()

	if next != done {
		time.Sleep(next)
	}

	return next
}

// Succeeded marks client request succeeded.
func (c *ConcurrentRetrier) Succeeded() {
	defer c.Unlock()
	c.Lock()
	c.failureCount = 0
	c.retrier.Reset()
}

// Failed marks client request failed because backend is busy.
func (c *ConcurrentRetrier) Failed() {
	defer c.Unlock()
	c.Lock()
	c.failureCount++
}

// NewConcurrentRetrier returns an instance of concurrent backoff retrier.
func NewConcurrentRetrier(retryPolicy RetryPolicy) *ConcurrentRetrier {
	retrier := NewRetrier(retryPolicy, SystemClock)
	return &ConcurrentRetrier{retrier: retrier}
}

// Retry function can be used to wrap any call with retry logic using the passed in policy
func Retry(ctx context.Context, operation Operation, policy RetryPolicy, isRetryable IsRetryable) error {
	var lastErr error
	var next time.Duration

	r := NewRetrier(policy, SystemClock)
	for {
		opErr := operation()
		if opErr == nil {
			// operation completed successfully. No need to retry.
			return nil
		}

		// Usually, after number of retry attempts, last attempt fails with DeadlineExceeded error.
		// It is not informative and actual error reason is in the error occurred on previous attempt.
		// Therefore, update lastErr only if it is not set (first attempt) or opErr is not a DeadlineExceeded error.
		// This lastErr is returned if retry attempts are exhausted.
		var errDeadlineExceeded *serviceerror.DeadlineExceeded
		if lastErr == nil || !(errors.Is(opErr, context.DeadlineExceeded) || errors.As(opErr, &errDeadlineExceeded)) {
			lastErr = opErr
		}

		if next = r.NextBackOff(); next == done {
			return lastErr
		}

		// Check if the error is retryable
		if isRetryable != nil && !isRetryable(opErr) {
			return lastErr
		}

		// check if ctx is done
		if ctxDone := ctx.Done(); ctxDone != nil {
			timer := time.NewTimer(next)
			select {
			case <-ctxDone:
				return lastErr
			case <-timer.C:
				continue
			}
		}

		// ctx is not cancellable
		time.Sleep(next)
	}
}

// IgnoreErrors can be used as IsRetryable handler for Retry function to exclude certain errors from the retry list
func IgnoreErrors(errorsToExclude []error) func(error) bool {
	return func(err error) bool {
		for _, errorToExclude := range errorsToExclude {
			if err == errorToExclude {
				return false
			}
		}

		return true
	}
}
