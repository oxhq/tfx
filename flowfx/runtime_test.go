package flowfx

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/oxhq/tfx/runfx"
)

type stubReporter struct {
	started   atomic.Int32
	updated   atomic.Int32
	completed atomic.Int32
	errored   atomic.Int32
}

func (s *stubReporter) Start(string, int) { s.started.Add(1) }
func (s *stubReporter) Update(int)        { s.updated.Add(1) }
func (s *stubReporter) Complete()         { s.completed.Add(1) }
func (s *stubReporter) Error(error)       { s.errored.Add(1) }

type stubScriptLogger struct {
	events []string
}

func (s *stubScriptLogger) LogStepStart(name, description string) {
	s.events = append(s.events, "step-start:"+name+":"+description)
}

func (s *stubScriptLogger) LogStepComplete(name string) {
	s.events = append(s.events, "step-complete:"+name)
}

func (s *stubScriptLogger) LogStepError(name string, err error) {
	s.events = append(s.events, "step-error:"+name+":"+err.Error())
}

func (s *stubScriptLogger) LogScriptStart(name string) {
	s.events = append(s.events, "script-start:"+name)
}

func (s *stubScriptLogger) LogScriptComplete(name string) {
	s.events = append(s.events, "script-complete:"+name)
}

func (s *stubScriptLogger) LogScriptError(name string, err error) {
	s.events = append(s.events, "script-error:"+name+":"+err.Error())
}

func TestSequenceRunCallsHooksAndExecutesInOrder(t *testing.T) {
	t.Parallel()

	var order []string
	started := false
	completed := false

	seq := NewSequenceBuilder().
		Name("release").
		OnStart(func(context.Context, string, error) { started = true }).
		OnComplete(func(context.Context, string, error) { completed = true }).
		Func("one", func(context.Context) error {
			order = append(order, "one")
			return nil
		}).
		Func("two", func(context.Context) error {
			order = append(order, "two")
			return nil
		}).
		Build()

	if err := seq.Run(context.Background()); err != nil {
		t.Fatalf("unexpected sequence error: %v", err)
	}
	if !started || !completed {
		t.Fatalf("expected start/complete hooks, got started=%v completed=%v", started, completed)
	}
	if strings.Join(order, ",") != "one,two" {
		t.Fatalf("unexpected step order: %v", order)
	}
}

func TestTaskTimeoutAndReporter(t *testing.T) {
	t.Parallel()

	reporter := &stubReporter{}
	task := NewTask(
		"timeout",
		func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		},
		WithTimeout(5*time.Millisecond),
		WithRetry(RetryConfig{MaxAttempts: 1, Delay: 0, Backoff: 1}),
		WithProgressReporter(reporter),
	)

	err := task.Execute(context.Background())
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected timeout error, got %v", err)
	}
	if reporter.started.Load() != 1 || reporter.completed.Load() != 1 ||
		reporter.errored.Load() != 1 {
		t.Fatalf(
			"unexpected reporter counts: started=%d completed=%d errored=%d",
			reporter.started.Load(),
			reporter.completed.Load(),
			reporter.errored.Load(),
		)
	}
}

func TestParallelRunFailFastCancelsOtherSteps(t *testing.T) {
	t.Parallel()

	canceled := make(chan struct{}, 1)
	failErr := errors.New("boom")

	p := NewParallelBuilder().
		FailFast(true).
		Func("wait", func(ctx context.Context) error {
			<-ctx.Done()
			canceled <- struct{}{}
			return ctx.Err()
		}).
		Func("fail", func(context.Context) error {
			return failErr
		}).
		Build()

	err := p.Run(context.Background())
	if err == nil || !errors.Is(err, failErr) {
		t.Fatalf("expected fail-fast error, got %v", err)
	}

	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("expected waiting step to observe cancellation")
	}
}

func TestScriptRunAggregatesErrorsAndSummary(t *testing.T) {
	t.Parallel()

	logger := &stubScriptLogger{}
	script := NewScriptBuilder().
		Name("deploy").
		Logger(logger).
		Build()
	script.AddStep(ScriptStep{
		Name:        "lint",
		Description: "static checks",
		Step:        StepFunc(func(context.Context) error { return errors.New("lint failed") }),
	})
	script.AddStep(ScriptStep{
		Name: "package",
		Step: StepFunc(func(context.Context) error { return nil }),
	})

	err := script.Run(context.Background())
	if err == nil {
		t.Fatal("expected aggregated script error")
	}
	if !strings.Contains(script.Summary(), "deploy") {
		t.Fatalf("expected summary to include script name, got %q", script.Summary())
	}
	if len(logger.events) == 0 || !strings.HasPrefix(logger.events[0], "script-start:deploy") {
		t.Fatalf("expected logger events, got %v", logger.events)
	}
}

func TestTreeRunTraverseAndNodeHelpers(t *testing.T) {
	t.Parallel()

	var order []string
	tree := NewTreeBuilder().Name("tree").Build()
	root := NewTreeNode("root", StepFunc(func(context.Context) error {
		order = append(order, "root")
		return nil
	}))
	child := NewTreeNode("child", StepFunc(func(context.Context) error {
		order = append(order, "child")
		return nil
	}))
	root.AddChild(child)
	tree.SetRoot(root)

	if err := tree.Run(context.Background()); err != nil {
		t.Fatalf("unexpected tree error: %v", err)
	}
	if strings.Join(order, ",") != "root,child" {
		t.Fatalf("unexpected execution order: %v", order)
	}
	if !root.IsRoot() || child.IsRoot() {
		t.Fatal("unexpected root detection")
	}
	if root.IsLeaf() || !child.IsLeaf() {
		t.Fatal("unexpected leaf detection")
	}
	if got := strings.Join(child.GetPath(), "/"); got != "root/child" {
		t.Fatalf("unexpected child path: %s", got)
	}

	var visited []string
	tree.Traverse(func(node *TreeNode, depth int) {
		visited = append(visited, node.Name)
	})
	if strings.Join(visited, ",") != "root,child" {
		t.Fatalf("unexpected traversal order: %v", visited)
	}
	if !strings.Contains(tree.String(), "root") || !strings.Contains(tree.String(), "child") {
		t.Fatalf("expected tree string representation, got %q", tree.String())
	}
}

func TestWizardRunMergesState(t *testing.T) {
	t.Parallel()

	wizard := New().
		Name("wizard").
		InitialState(map[string]any{"env": "dev"}).
		Func("set-version", func(_ context.Context, input map[string]any) (map[string]any, error) {
			if input["env"] != "dev" {
				t.Fatalf("unexpected initial state: %#v", input)
			}
			return map[string]any{"version": "v0.1.0"}, nil
		}).
		Func("set-status", func(_ context.Context, input map[string]any) (map[string]any, error) {
			if input["version"] != "v0.1.0" {
				t.Fatalf("expected previous output in state: %#v", input)
			}
			return map[string]any{"status": "ready"}, nil
		}).
		Build()

	if err := wizard.Run(context.Background()); err != nil {
		t.Fatalf("unexpected wizard error: %v", err)
	}

	state := wizard.GetState()
	if state["env"] != "dev" || state["version"] != "v0.1.0" || state["status"] != "ready" {
		t.Fatalf("unexpected wizard state: %#v", state)
	}
}

func TestBranchAndRunnerSuccessPath(t *testing.T) {
	t.Parallel()

	calls := 0
	trueFlow := NewSequenceBuilder().
		Func("true", func(context.Context) error {
			calls++
			return nil
		}).
		Build()

	branch := NewBranch(func(context.Context) (bool, error) { return true, nil }).When(trueFlow)
	runner := NewRunnerBuilder().Name("runner").HandleSigInt(false).Flow(branch).Build()
	runner.ttyInfo = &runfx.TTYInfo{IsTTY: true}

	if err := runner.Start(context.Background()); err != nil {
		t.Fatalf("unexpected runner error: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected true branch to run once, got %d", calls)
	}
	if runner.IsRunning() {
		t.Fatal("runner should not remain running after completion")
	}
	if !runner.IsInteractive() && runner.GetTTYInfo().IsTTY {
		t.Fatal("interactive getters disagreed")
	}
}

func TestFlowErrorAndMultiErrorHelpers(t *testing.T) {
	t.Parallel()

	base := errors.New("base")
	flowErr := NewFlowErrorWithAttempt("deploy", "lint", base, 1)
	if !errors.Is(flowErr, base) {
		t.Fatalf("expected flow error to wrap base error, got %v", flowErr)
	}

	multi := NewMultiError()
	if multi.ToError() != nil {
		t.Fatal("expected empty multi-error to convert to nil")
	}
	multi.Add(flowErr)
	if !multi.HasErrors() {
		t.Fatal("expected multi-error to contain error")
	}
	if multi.Unwrap() == nil || multi.Error() == "" {
		t.Fatalf(
			"expected unwrap/error output, got unwrap=%v error=%q",
			multi.Unwrap(),
			multi.Error(),
		)
	}
}
