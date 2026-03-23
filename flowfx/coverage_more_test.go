package flowfx

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestFlowfxBuilderSettersAndConvenienceHelpers(t *testing.T) {
	t.Parallel()

	startHook := func(context.Context, string, error) {}
	completeHook := func(context.Context, string, error) {}
	errorHook := func(context.Context, string, error) {}

	wizardStep := NewWizardStepFunc("collect-input", func(
		_ context.Context,
		input map[string]any,
	) (map[string]any, error) {
		return input, nil
	})
	if wizardStep.Label() != "collect-input" {
		t.Fatalf("unexpected wizard step label: %q", wizardStep.Label())
	}

	mapFlow := NewMapFlowBuilder().
		OnStart(startHook).
		OnComplete(completeHook).
		OnError(errorHook).
		Step("one", StepFunc(func(context.Context) error { return nil })).
		Build()
	if mapFlow.onStart == nil || mapFlow.onComplete == nil || mapFlow.onError == nil {
		t.Fatal("expected mapflow hooks to be preserved")
	}
	if len(mapFlow.Steps()) != 1 {
		t.Fatalf("expected one step copy from mapflow, got %d", len(mapFlow.Steps()))
	}

	parallel := NewParallelBuilder().
		OnStart(startHook).
		OnComplete(completeHook).
		OnError(errorHook).
		Build()
	if parallel.onStart == nil || parallel.onComplete == nil || parallel.onError == nil {
		t.Fatal("expected parallel hooks to be preserved")
	}

	script := NewScriptBuilder().
		OnStart(startHook).
		OnComplete(completeHook).
		OnError(errorHook).
		Build()
	if script.onStart == nil || script.onComplete == nil || script.onError == nil {
		t.Fatal("expected script hooks to be preserved")
	}

	root := NewTreeNode("root", StepFunc(func(context.Context) error { return nil }))
	tree := NewTreeBuilder().
		OnStart(startHook).
		OnComplete(completeHook).
		OnError(errorHook).
		Root(root).
		Build()
	if tree.root != root || tree.onStart == nil || tree.onComplete == nil || tree.onError == nil {
		t.Fatal("expected tree builder to preserve root and hooks")
	}

	falseRan := false
	branch := NewBranch(func(context.Context) (bool, error) { return false, nil }).
		Else(NewSequenceBuilder().Func("false", func(context.Context) error {
			falseRan = true
			return nil
		}).Build())
	if err := branch.Run(context.Background()); err != nil {
		t.Fatalf("unexpected branch error: %v", err)
	}
	if !falseRan {
		t.Fatal("expected false branch to run")
	}
}

func TestFlowfxErrorFormattingAndTaskHooks(t *testing.T) {
	t.Parallel()

	base := errors.New("boom")
	if got := NewFlowError("deploy", "", base).Error(); !strings.Contains(
		got,
		"flow deploy: boom",
	) {
		t.Fatalf("unexpected flow error string without step: %q", got)
	}
	if got := (&MultiError{Errors: []error{base}}).Error(); got != "boom" {
		t.Fatalf("unexpected single-error output: %q", got)
	}

	started := false
	completed := false
	errored := false

	okTask := NewTask(
		"ok",
		func(context.Context) error { return nil },
		WithRetry(RetryConfig{MaxAttempts: 1}),
		WithOnStart(func(context.Context, string, error) { started = true }),
		WithOnComplete(func(context.Context, string, error) { completed = true }),
	)
	if err := okTask.Execute(context.Background()); err != nil {
		t.Fatalf("unexpected task success error: %v", err)
	}
	if !started || !completed {
		t.Fatalf("expected success hooks to fire, got started=%v completed=%v", started, completed)
	}

	failTask := NewTask(
		"fail",
		func(context.Context) error { return base },
		WithRetry(RetryConfig{MaxAttempts: 1}),
		WithOnError(func(context.Context, string, error) { errored = true }),
	)
	err := failTask.Execute(context.Background())
	if err == nil || !errors.Is(err, base) {
		t.Fatalf("expected failing task to wrap base error, got %v", err)
	}
	if !errored {
		t.Fatal("expected error hook to fire")
	}
}

func TestFlowfxSequenceBranchAndTreeErrorPaths(t *testing.T) {
	t.Parallel()

	boom := errors.New("boom")

	seqHookCalled := false
	seqErr := NewSequenceBuilder().
		Name("release").
		OnError(func(_ context.Context, label string, err error) {
			seqHookCalled = label == "release" && err != nil
		}).
		Task(NewTask("lint", func(context.Context) error { return boom }, WithRetry(RetryConfig{MaxAttempts: 1}))).
		Run(context.Background())
	if seqErr == nil || !errors.Is(seqErr, boom) {
		t.Fatalf("expected sequence error to wrap boom, got %v", seqErr)
	}
	if !strings.Contains(seqErr.Error(), "step lint") {
		t.Fatalf("expected labeled step in sequence error, got %q", seqErr.Error())
	}
	if !seqHookCalled {
		t.Fatal("expected sequence onError hook to fire")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceledErr := NewSequenceBuilder().
		Name("cancel-seq").
		Step(StepFunc(func(context.Context) error {
			t.Fatal("canceled sequence should not execute steps")
			return nil
		})).
		Run(ctx)
	if canceledErr == nil || !errors.Is(canceledErr, context.Canceled) {
		t.Fatalf("expected canceled sequence error, got %v", canceledErr)
	}

	branchHookCalled := false
	branchErr := NewBranch(func(context.Context) (bool, error) {
		return false, boom
	}, BranchConfig{
		Name:    "branch",
		OnError: func(_ context.Context, label string, err error) { branchHookCalled = label == "branch" && err != nil },
	}).Run(context.Background())
	if branchErr == nil || !errors.Is(branchErr, boom) {
		t.Fatalf("expected condition error to wrap boom, got %v", branchErr)
	}
	if !branchHookCalled {
		t.Fatal("expected branch onError hook to fire")
	}

	treeHookCalled := false
	treeErr := NewTreeBuilder().
		Name("tree").
		OnError(func(_ context.Context, label string, err error) {
			treeHookCalled = label == "tree" && err != nil
		}).
		Root(NewTreeNode("root", StepFunc(func(context.Context) error { return boom }))).
		Run(context.Background())
	if treeErr == nil || !errors.Is(treeErr, boom) {
		t.Fatalf("expected tree error to wrap boom, got %v", treeErr)
	}
	if !treeHookCalled {
		t.Fatal("expected tree onError hook to fire")
	}
}

func TestMustPanicsOnInvalidMultipathInput(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("expected invalid constructor input to panic")
		}
	}()

	_ = NewSequence("bad")
}
