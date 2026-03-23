package flowfx

import (
	"context"
	"errors"
	"testing"
)

func TestFlowRunnerStartNilFlow(t *testing.T) {
	var runner *FlowRunner

	err := runner.Start(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrNilFlow) {
		t.Fatalf("expected ErrNilFlow, got %v", err)
	}
}

func TestTaskExecuteNilRun(t *testing.T) {
	task := NewTask("missing", nil)

	err := task.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidTask) {
		t.Fatalf("expected ErrInvalidTask, got %v", err)
	}
}

func TestTaskExecuteReturnsLastRetryError(t *testing.T) {
	firstErr := errors.New("first failure")
	lastErr := errors.New("last failure")
	calls := 0

	task := NewTask("retry", func(context.Context) error {
		calls++
		if calls == 1 {
			return firstErr
		}
		return lastErr
	}, WithRetry(RetryConfig{
		MaxAttempts: 2,
		Delay:       0,
		Backoff:     1,
	}))

	err := task.Execute(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, lastErr) {
		t.Fatalf("expected final error to wrap last failure, got %v", err)
	}
	if errors.Is(err, ErrRetryExhausted) {
		t.Fatalf("did not expect ErrRetryExhausted, got %v", err)
	}

	var flowErr *FlowError
	if !errors.As(err, &flowErr) {
		t.Fatalf("expected FlowError, got %T", err)
	}
	if flowErr.Attempt != 1 {
		t.Fatalf("expected attempt 1, got %d", flowErr.Attempt)
	}
}

func TestMapFlowAddStepReplacesWithoutDuplicateOrder(t *testing.T) {
	mf := NewMapFlow()
	firstCalls := 0
	secondCalls := 0

	mf.AddFunc("step", "first", func(context.Context) error {
		firstCalls++
		return nil
	})
	mf.AddFunc("step", "second", func(context.Context) error {
		secondCalls++
		return nil
	})

	order := mf.Order()
	if len(order) != 1 || order[0] != "step" {
		t.Fatalf("expected unique order entry, got %#v", order)
	}

	if err := mf.Run(context.Background()); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if firstCalls != 0 {
		t.Fatalf("expected replaced step to be ignored, firstCalls=%d", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("expected replacement step to run once, secondCalls=%d", secondCalls)
	}
}

func TestMapFlowBuilderReplacesWithoutDuplicateOrder(t *testing.T) {
	builder := NewMapFlowBuilder()
	firstCalls := 0
	secondCalls := 0

	builder.Func("step", "first", func(context.Context) error {
		firstCalls++
		return nil
	})
	builder.Func("step", "second", func(context.Context) error {
		secondCalls++
		return nil
	})

	mf := builder.Build()
	order := mf.Order()
	if len(order) != 1 || order[0] != "step" {
		t.Fatalf("expected unique order entry, got %#v", order)
	}

	if err := mf.Run(context.Background()); err != nil {
		t.Fatalf("unexpected run error: %v", err)
	}
	if firstCalls != 0 {
		t.Fatalf("expected replaced step to be ignored, firstCalls=%d", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("expected replacement step to run once, secondCalls=%d", secondCalls)
	}
}

func TestTryConstructorsRejectInvalidMultipathArgs(t *testing.T) {
	tests := []struct {
		name string
		fn   func() error
	}{
		{name: "sequence", fn: func() error { _, err := TryNewSequence("bad"); return err }},
		{name: "parallel", fn: func() error { _, err := TryNewParallel("bad"); return err }},
		{name: "mapflow", fn: func() error { _, err := TryNewMapFlow("bad"); return err }},
		{name: "script", fn: func() error { _, err := TryNewScript("bad"); return err }},
		{name: "tree", fn: func() error { _, err := TryNewTree("bad"); return err }},
		{name: "wizard", fn: func() error { _, err := TryNewWizard("bad"); return err }},
		{name: "runner", fn: func() error { _, err := TryNewRunner(nil, "bad"); return err }},
		{
			name: "branch",
			fn: func() error {
				_, err := TryNewBranch(
					func(context.Context) (bool, error) { return true, nil },
					"bad",
				)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fn(); err == nil {
				t.Fatal("expected invalid config error")
			}
		})
	}
}
