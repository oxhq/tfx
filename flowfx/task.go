package flowfx

import (
	"context"
	"time"
)

// Task represents a single unit of work with metadata and execution options.
type Task struct {
	Label      string
	Run        func(ctx context.Context) error
	Retry      RetryConfig
	Timeout    time.Duration
	Reporter   ProgressReporter
	OnStart    Hook
	OnComplete Hook
	OnError    Hook
}

// TaskOption is a functional option for configuring a Task.
type TaskOption func(*Task)

// NewTask creates a new Task with the provided label and function.
func NewTask(label string, fn func(ctx context.Context) error, opts ...TaskOption) *Task {
	task := &Task{
		Label:   label,
		Run:     fn,
		Retry:   DefaultRetryConfig(),
		Timeout: 0, // No timeout by default
	}

	for _, opt := range opts {
		opt(task)
	}

	return task
}

// WithRetry sets the retry configuration for the task.
func WithRetry(config RetryConfig) TaskOption {
	return func(t *Task) {
		t.Retry = config
	}
}

// WithTimeout sets the timeout for the task.
func WithTimeout(timeout time.Duration) TaskOption {
	return func(t *Task) {
		t.Timeout = timeout
	}
}

// WithProgressReporter sets the progress reporter for the task.
func WithProgressReporter(reporter ProgressReporter) TaskOption {
	return func(t *Task) {
		t.Reporter = reporter
	}
}

// WithOnStart sets a hook to be called when the task starts.
func WithOnStart(hook Hook) TaskOption {
	return func(t *Task) {
		t.OnStart = hook
	}
}

// WithOnComplete sets a hook to be called when the task completes successfully.
func WithOnComplete(hook Hook) TaskOption {
	return func(t *Task) {
		t.OnComplete = hook
	}
}

// WithOnError sets a hook to be called when the task encounters an error.
func WithOnError(hook Hook) TaskOption {
	return func(t *Task) {
		t.OnError = hook
	}
}

// Execute implements the Step interface for Task.
func (t *Task) Execute(ctx context.Context) error {
	if t == nil {
		return NewFlowError("task", "", ErrInvalidTask)
	}

	// Create timeout context if specified
	execCtx := ctx
	var cancel context.CancelFunc
	if t.Timeout > 0 {
		execCtx, cancel = context.WithTimeout(ctx, t.Timeout)
		defer cancel()
	}

	// Start progress reporting
	if t.Reporter != nil {
		t.Reporter.Start(t.Label, 1)
		defer func() {
			t.Reporter.Complete()
		}()
	}

	// Call start hook
	if t.OnStart != nil {
		t.OnStart(execCtx, t.Label, nil)
	}

	if t.Run == nil {
		err := NewFlowErrorWithAttempt("task", t.Label, ErrInvalidTask, 0)
		if t.OnError != nil {
			t.OnError(execCtx, t.Label, err)
		}
		if t.Reporter != nil {
			t.Reporter.Error(err)
		}
		return err
	}

	// Execute with retry logic
	var lastErr error
	delay := t.Retry.Delay
	attempts := t.Retry.MaxAttempts
	if attempts < 1 {
		attempts = 1
	}

	for attempt := 0; attempt < attempts; attempt++ {
		// Check for cancellation before each attempt
		select {
		case <-execCtx.Done():
			if execCtx.Err() == context.DeadlineExceeded {
				lastErr = ErrTimeout
			} else {
				lastErr = ErrCanceled
			}
			if t.OnError != nil {
				t.OnError(execCtx, t.Label, lastErr)
			}
			if t.Reporter != nil {
				t.Reporter.Error(lastErr)
			}
			return NewFlowErrorWithAttempt("task", t.Label, lastErr, attempt)
		default:
		}

		// Execute the task
		err := t.Run(execCtx)
		if err == nil {
			// Success
			if t.OnComplete != nil {
				t.OnComplete(execCtx, t.Label, nil)
			}
			if t.Reporter != nil {
				t.Reporter.Update(1)
			}
			return nil
		}

		lastErr = err

		// If this is the last attempt, don't wait
		if attempt == attempts-1 {
			break
		}

		// Wait before retry with exponential backoff
		select {
		case <-execCtx.Done():
			if execCtx.Err() == context.DeadlineExceeded {
				lastErr = ErrTimeout
			} else {
				lastErr = ErrCanceled
			}
			if t.OnError != nil {
				t.OnError(execCtx, t.Label, lastErr)
			}
			if t.Reporter != nil {
				t.Reporter.Error(lastErr)
			}
			return NewFlowErrorWithAttempt("task", t.Label, lastErr, attempt)
		case <-time.After(delay):
			delay = time.Duration(float64(delay) * t.Retry.Backoff)
		}
	}

	// All retries exhausted
	if lastErr == nil {
		lastErr = ErrRetryExhausted
	}
	finalErr := NewFlowErrorWithAttempt("task", t.Label, lastErr, attempts-1)
	if t.OnError != nil {
		t.OnError(execCtx, t.Label, finalErr)
	}
	if t.Reporter != nil {
		t.Reporter.Error(finalErr)
	}

	return finalErr
}
