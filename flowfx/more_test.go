package flowfx

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/oxhq/tfx/runfx"
)

type blockingFlow struct {
	started chan struct{}
	done    chan struct{}
}

func (f *blockingFlow) Run(ctx context.Context) error {
	close(f.started)
	<-ctx.Done()
	close(f.done)
	return ctx.Err()
}

func TestSequenceParallelAndMapFlowConvenienceMethods(t *testing.T) {
	t.Parallel()

	var seqOrder []string
	seq := NewSequence().
		Add(StepFunc(func(context.Context) error {
			seqOrder = append(seqOrder, "step")
			return nil
		})).
		AddTask(NewTask("task", func(context.Context) error {
			seqOrder = append(seqOrder, "task")
			return nil
		})).
		AddFunc("func", func(context.Context) error {
			seqOrder = append(seqOrder, "func")
			return nil
		})
	if seq.Len() != 3 || len(seq.Steps()) != 3 {
		t.Fatalf("expected 3 sequence steps, got len=%d copy=%d", seq.Len(), len(seq.Steps()))
	}
	if err := seq.Run(context.Background()); err != nil {
		t.Fatalf("unexpected sequence error: %v", err)
	}
	if strings.Join(seqOrder, ",") != "step,task,func" {
		t.Fatalf("unexpected sequence order: %v", seqOrder)
	}

	var seqBuilderOrder []string
	seqBuilderErr := NewSequenceBuilder().
		Name("builder-sequence").
		Step(StepFunc(func(context.Context) error {
			seqBuilderOrder = append(seqBuilderOrder, "one")
			return nil
		})).
		Steps(StepFunc(func(context.Context) error {
			seqBuilderOrder = append(seqBuilderOrder, "two")
			return nil
		})).
		Task(NewTask("three", func(context.Context) error {
			seqBuilderOrder = append(seqBuilderOrder, "three")
			return nil
		})).
		Run(context.Background())
	if seqBuilderErr != nil {
		t.Fatalf("unexpected sequence builder error: %v", seqBuilderErr)
	}
	if strings.Join(seqBuilderOrder, ",") != "one,two,three" {
		t.Fatalf("unexpected sequence builder order: %v", seqBuilderOrder)
	}

	var parSeen []string
	par := NewParallel().
		Add(StepFunc(func(context.Context) error {
			parSeen = append(parSeen, "step")
			return nil
		})).
		AddTask(NewTask("task", func(context.Context) error {
			parSeen = append(parSeen, "task")
			return nil
		})).
		AddFunc("func", func(context.Context) error {
			parSeen = append(parSeen, "func")
			return nil
		})
	if par.Len() != 3 || len(par.Steps()) != 3 {
		t.Fatalf("expected 3 parallel steps, got len=%d copy=%d", par.Len(), len(par.Steps()))
	}
	if err := par.Run(context.Background()); err != nil {
		t.Fatalf("unexpected parallel error: %v", err)
	}

	var parBuilderCount int
	parBuilderErr := NewParallelBuilder().
		Name("builder-parallel").
		FailFast(false).
		Step(StepFunc(func(context.Context) error {
			parBuilderCount++
			return nil
		})).
		Steps(StepFunc(func(context.Context) error {
			parBuilderCount++
			return nil
		})).
		Task(NewTask("third", func(context.Context) error {
			parBuilderCount++
			return nil
		})).
		Run(context.Background())
	if parBuilderErr != nil {
		t.Fatalf("unexpected parallel builder error: %v", parBuilderErr)
	}
	if parBuilderCount != 3 {
		t.Fatalf("expected 3 parallel builder executions, got %d", parBuilderCount)
	}

	mf := NewMapFlow().
		AddTask("task", NewTask("task", func(context.Context) error { return nil })).
		AddFunc("func", "func", func(context.Context) error { return nil }).
		SetConfig("env", "dev").
		SetOrder([]string{"func", "task", "func"})
	if value, ok := mf.GetConfig("env"); !ok || value != "dev" {
		t.Fatalf("expected mapflow config env=dev, got %v %v", value, ok)
	}
	if got := mf.Config(); got["env"] != "dev" {
		t.Fatalf("expected config copy to contain env, got %#v", got)
	}
	if got := mf.Order(); len(got) != 2 || got[0] != "func" || got[1] != "task" {
		t.Fatalf("unexpected deduped order: %#v", got)
	}
	if err := mf.Run(context.Background()); err != nil {
		t.Fatalf("unexpected mapflow error: %v", err)
	}

	payload, err := mf.ToJSON()
	if err != nil {
		t.Fatalf("unexpected json error: %v", err)
	}
	restored := NewMapFlow()
	if err := restored.FromJSON(payload); err != nil {
		t.Fatalf("unexpected restore error: %v", err)
	}
	if got := restored.Order(); len(got) != 2 || got[0] != "func" || got[1] != "task" {
		t.Fatalf("unexpected restored order: %#v", got)
	}
	if got := restored.Config(); got["env"] != "dev" {
		t.Fatalf("unexpected restored config: %#v", got)
	}

	builderCount := 0
	mapBuilderErr := NewMapFlowBuilder().
		Name("builder-map").
		Config(map[string]any{"feature": "on"}).
		Step("one", StepFunc(func(context.Context) error {
			builderCount++
			return nil
		})).
		Task("two", NewTask("two", func(context.Context) error {
			builderCount++
			return nil
		})).
		Order([]string{"two", "one", "two"}).
		Run(context.Background())
	if mapBuilderErr != nil {
		t.Fatalf("unexpected mapflow builder error: %v", mapBuilderErr)
	}
	if builderCount != 2 {
		t.Fatalf("expected 2 mapflow builder executions, got %d", builderCount)
	}
}

func TestScriptTreeAndWizardBuilderHelpers(t *testing.T) {
	t.Parallel()

	logger := &stubScriptLogger{}
	script := NewScript().
		AddSimpleStep("one", StepFunc(func(context.Context) error { return nil })).
		AddTask("two", NewTask("two", func(context.Context) error { return nil })).
		AddFunc("three", "three", func(context.Context) error { return nil }).
		AddCriticalStep("four", StepFunc(func(context.Context) error { return nil }))
	script.logger = logger
	if script.Len() != 4 || len(script.Steps()) != 4 {
		t.Fatalf("expected 4 script steps, got len=%d copy=%d", script.Len(), len(script.Steps()))
	}
	if summary := script.Summary(); !strings.Contains(summary, "critical") ||
		!strings.Contains(summary, "four") {
		t.Fatalf("unexpected script summary: %q", summary)
	}
	if err := script.Run(context.Background()); err != nil {
		t.Fatalf("unexpected script run error: %v", err)
	}

	builderLogger := &stubScriptLogger{}
	scriptBuilderErr := NewScriptBuilder().
		Name("builder-script").
		Logger(builderLogger).
		Step(ScriptStep{Name: "plain", Step: StepFunc(func(context.Context) error { return nil })}).
		SimpleStep("simple", StepFunc(func(context.Context) error { return nil })).
		Task("task", NewTask("task", func(context.Context) error { return nil })).
		Func("func", "func", func(context.Context) error { return nil }).
		CriticalStep("critical", StepFunc(func(context.Context) error { return nil })).
		SilentStep("silent", StepFunc(func(context.Context) error { return nil })).
		Run(context.Background())
	if scriptBuilderErr != nil {
		t.Fatalf("unexpected script builder error: %v", scriptBuilderErr)
	}
	if len(builderLogger.events) == 0 {
		t.Fatal("expected script builder logger events")
	}

	var treeOrder []string
	tree := NewTree().CreateRoot("root", StepFunc(func(context.Context) error {
		treeOrder = append(treeOrder, "root")
		return nil
	}))
	root := tree.GetRoot()
	root.AddChildFunc("child-func", "child-func", func(context.Context) error {
		treeOrder = append(treeOrder, "child-func")
		return nil
	})
	root.AddChildTask("child-task", NewTask("child-task", func(context.Context) error {
		treeOrder = append(treeOrder, "child-task")
		return nil
	}))
	if len(root.GetChildren()) != 2 {
		t.Fatalf("expected 2 root children, got %d", len(root.GetChildren()))
	}
	if err := tree.Run(context.Background()); err != nil {
		t.Fatalf("unexpected tree run error: %v", err)
	}
	if strings.Join(treeOrder, ",") != "root,child-func,child-task" {
		t.Fatalf("unexpected tree order: %v", treeOrder)
	}

	node := NewTreeNodeBuilder("branch", StepFunc(func(context.Context) error { return nil })).
		Child(NewTreeNode("leaf", StepFunc(func(context.Context) error { return nil }))).
		ChildFunc("generated-func", "generated-func", func(context.Context) error { return nil }).
		ChildTask("generated-task", NewTask("generated-task", func(context.Context) error { return nil })).
		Build()
	if len(node.GetChildren()) != 3 {
		t.Fatalf("expected 3 built children, got %d", len(node.GetChildren()))
	}
	if err := NewTreeBuilder().
		Name("func-tree").
		RootFunc("root-func", "root-func", func(context.Context) error { return nil }).
		Run(context.Background()); err != nil {
		t.Fatalf("unexpected tree builder func error: %v", err)
	}
	if err := NewTreeBuilder().
		Name("task-tree").
		RootTask("root-task", NewTask("root-task", func(context.Context) error { return nil })).
		Run(context.Background()); err != nil {
		t.Fatalf("unexpected tree builder task error: %v", err)
	}

	wizard := NewWizard().
		AddStep(NewWizardStepFunc("one", func(_ context.Context, input map[string]any) (map[string]any, error) {
			if input["seed"] != "start" {
				t.Fatalf("unexpected wizard seed state: %#v", input)
			}
			return map[string]any{"one": 1}, nil
		})).
		AddFunc("two", func(_ context.Context, input map[string]any) (map[string]any, error) {
			if input["one"] != 1 {
				t.Fatalf("expected state merge before second step, got %#v", input)
			}
			return map[string]any{"two": 2}, nil
		}).
		SetState("seed", "start")
	if wizard.Len() != 2 || len(wizard.Steps()) != 2 {
		t.Fatalf("expected 2 wizard steps, got len=%d copy=%d", wizard.Len(), len(wizard.Steps()))
	}
	if err := wizard.Run(context.Background()); err != nil {
		t.Fatalf("unexpected wizard run error: %v", err)
	}
	state := wizard.GetState()
	if state["seed"] != "start" || state["one"] != 1 || state["two"] != 2 {
		t.Fatalf("unexpected wizard state: %#v", state)
	}

	started := false
	completed := false
	builderErr := New().
		Name("wizard-builder").
		OnStart(func(context.Context, string, error) { started = true }).
		OnComplete(func(context.Context, string, error) { completed = true }).
		OnError(func(context.Context, string, error) { t.Fatal("did not expect wizard error hook") }).
		InitialState(map[string]any{"base": 1}).
		Step(NewWizardStepFunc("builder-step", func(_ context.Context, input map[string]any) (map[string]any, error) {
			if input["base"] != 1 {
				t.Fatalf("unexpected wizard builder input: %#v", input)
			}
			return map[string]any{"done": true}, nil
		})).
		Run(context.Background())
	if builderErr != nil {
		t.Fatalf("unexpected wizard builder error: %v", builderErr)
	}
	if !started || !completed {
		t.Fatalf("expected wizard hooks, got started=%v completed=%v", started, completed)
	}
}

func TestBranchRunnerAndEnvironmentHelpers(t *testing.T) {
	falseCalls := 0
	if err := NewBranch(nil).Run(context.Background()); err == nil ||
		!errors.Is(err, ErrInvalidCondition) {
		t.Fatalf("expected invalid condition error, got %v", err)
	}

	branchErr := NewBranchBuilder(func(context.Context) (bool, error) { return false, nil }).
		Name("feature-switch").
		OnStart(func(context.Context, string, error) {}).
		OnComplete(func(context.Context, string, error) {}).
		OnError(func(context.Context, string, error) { t.Fatal("did not expect branch error") }).
		When(NewSequenceBuilder().Func("true", func(context.Context) error { t.Fatal("true path should not run"); return nil }).Build()).
		Else(NewSequenceBuilder().Func("false", func(context.Context) error {
			falseCalls++
			return nil
		}).Build()).
		Run(context.Background())
	if branchErr != nil {
		t.Fatalf("unexpected branch builder error: %v", branchErr)
	}
	if falseCalls != 1 {
		t.Fatalf("expected false path to run once, got %d", falseCalls)
	}

	simpleFlow := NewSequenceBuilder().
		Func("single", func(context.Context) error { return nil }).
		Build()
	if err := RunFlow(context.Background(), simpleFlow); err != nil {
		t.Fatalf("unexpected RunFlow error: %v", err)
	}
	if err := RunFlowWithInterrupts(context.Background(), simpleFlow); err != nil {
		t.Fatalf("unexpected RunFlowWithInterrupts error: %v", err)
	}
	if err := RunFlowNonInteractive(context.Background(), simpleFlow); err != nil {
		t.Fatalf("unexpected RunFlowNonInteractive error: %v", err)
	}

	ttyInfo := GetEnvironmentInfo()
	if ttyInfo != runfx.DetectTTY() {
		t.Fatalf(
			"expected environment info to match DetectTTY, got %#v vs %#v",
			ttyInfo,
			runfx.DetectTTY(),
		)
	}
	if IsInteractiveEnvironment() != ttyInfo.IsTTY {
		t.Fatalf(
			"expected interactive helper to match tty info, got %v vs %v",
			IsInteractiveEnvironment(),
			ttyInfo.IsTTY,
		)
	}

	if err := NewRunner(simpleFlow).Stop(); err == nil || !errors.Is(err, ErrNotRunning) {
		t.Fatalf("expected ErrNotRunning from idle runner, got %v", err)
	}

	blocking := &blockingFlow{
		started: make(chan struct{}),
		done:    make(chan struct{}),
	}
	stopped := false
	runner := NewRunnerBuilder().
		Name("blocking").
		OnStart(func(context.Context, string, error) {}).
		OnComplete(func(context.Context, string, error) {}).
		OnError(func(context.Context, string, error) {}).
		OnStop(func(context.Context, string, error) { stopped = true }).
		HandleSigInt(false).
		Flow(blocking).
		Build()
	runner.ttyInfo = &runfx.TTYInfo{IsTTY: true}
	if runner.GetFlow() != blocking {
		t.Fatal("expected runner to expose configured flow")
	}
	if runner.GetTTYInfo().IsTTY != runner.IsInteractive() {
		t.Fatal("expected interactive helpers to agree")
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- runner.Start(context.Background())
	}()

	select {
	case <-blocking.started:
	case <-time.After(time.Second):
		t.Fatal("runner did not start blocking flow")
	}
	if !runner.IsRunning() {
		t.Fatal("expected runner to report running state")
	}
	if err := runner.Stop(); err != nil {
		t.Fatalf("unexpected runner stop error: %v", err)
	}
	select {
	case <-blocking.done:
	case <-time.After(time.Second):
		t.Fatal("runner stop did not cancel blocking flow")
	}
	if !stopped {
		t.Fatal("expected stop hook to run")
	}
	if err := <-errCh; err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("expected canceled start error, got %v", err)
	}
}
