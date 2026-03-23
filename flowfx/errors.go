package flowfx

import (
	"errors"
	"fmt"
)

// Common flow execution errors
var (
	// ErrCanceled indicates the flow was canceled by context
	ErrCanceled = errors.New("flow execution canceled")

	// ErrTimeout indicates the flow exceeded its timeout
	ErrTimeout = errors.New("flow execution timeout")

	// ErrRetryExhausted indicates all retry attempts have been used
	ErrRetryExhausted = errors.New("retry attempts exhausted")

	// ErrEmptyFlow indicates an attempt to run an empty flow
	ErrEmptyFlow = errors.New("cannot run empty flow")

	// ErrNilFlow indicates an attempt to run a nil flow
	ErrNilFlow = errors.New("flow is nil")

	// ErrInvalidTask indicates a task has no runnable function
	ErrInvalidTask = errors.New("task run function is nil")

	// ErrInvalidCondition indicates a branching condition is invalid
	ErrInvalidCondition = errors.New("invalid branching condition")

	// ErrNonInteractive indicates an operation requires interaction but none is available.
	ErrNonInteractive = errors.New(
		"operation requires interaction but running in non-interactive mode",
	)

	// ErrAlreadyRunning indicates an attempt to start a runner that's already running
	ErrAlreadyRunning = errors.New("runner is already running")

	// ErrNotRunning indicates an attempt to stop a runner that's not running
	ErrNotRunning = errors.New("runner is not running")
)

// FlowError represents an error that occurred during flow execution.
// It includes context about where in the flow the error occurred.
type FlowError struct {
	Flow    string // Name or type of the flow
	Step    string // Name or identifier of the step
	Err     error  // The underlying error
	Attempt int    // Which retry attempt failed (0 for first attempt)
}

// Error implements the error interface.
func (e *FlowError) Error() string {
	if e.Step != "" {
		if e.Attempt > 0 {
			return fmt.Sprintf(
				"flow %s, step %s (attempt %d): %v",
				e.Flow,
				e.Step,
				e.Attempt+1,
				e.Err,
			)
		}
		return fmt.Sprintf("flow %s, step %s: %v", e.Flow, e.Step, e.Err)
	}
	return fmt.Sprintf("flow %s: %v", e.Flow, e.Err)
}

// Unwrap returns the underlying error.
func (e *FlowError) Unwrap() error {
	return e.Err
}

// Is checks if the error matches the target error.
func (e *FlowError) Is(target error) bool {
	return errors.Is(e.Err, target)
}

// NewFlowError creates a new FlowError.
func NewFlowError(flow, step string, err error) *FlowError {
	return &FlowError{
		Flow: flow,
		Step: step,
		Err:  err,
	}
}

// NewFlowErrorWithAttempt creates a new FlowError with retry attempt information.
func NewFlowErrorWithAttempt(flow, step string, err error, attempt int) *FlowError {
	return &FlowError{
		Flow:    flow,
		Step:    step,
		Err:     err,
		Attempt: attempt,
	}
}

// MultiError represents multiple errors that occurred during parallel execution.
type MultiError struct {
	Errors []error
}

// Error implements the error interface.
func (m *MultiError) Error() string {
	if len(m.Errors) == 0 {
		return "no errors"
	}
	if len(m.Errors) == 1 {
		return m.Errors[0].Error()
	}
	return fmt.Sprintf("multiple errors: %d failures", len(m.Errors))
}

// Unwrap returns the first error for compatibility with errors.Is/As.
func (m *MultiError) Unwrap() error {
	if len(m.Errors) == 0 {
		return nil
	}
	return m.Errors[0]
}

// Add appends an error to the MultiError.
func (m *MultiError) Add(err error) {
	if err != nil {
		m.Errors = append(m.Errors, err)
	}
}

// HasErrors returns true if there are any errors.
func (m *MultiError) HasErrors() bool {
	return len(m.Errors) > 0
}

// ToError returns the MultiError as an error, or nil if no errors.
func (m *MultiError) ToError() error {
	if !m.HasErrors() {
		return nil
	}
	return m
}

// NewMultiError creates a new MultiError.
func NewMultiError() *MultiError {
	return &MultiError{
		Errors: make([]error, 0),
	}
}
