package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/oxhq/tfx/runfx"
)

type runOptions struct {
	flow    string
	lane    string
	run     bool
	json    bool
	quiet   bool
	version bool
}

func main() {
	if err := runWithArgsEnv(os.Stdout, os.Stdin, os.Args[1:], os.LookupEnv); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "tfx: %v\n", err)
		os.Exit(1)
	}
}

func run(output io.Writer, input io.Reader) error {
	return runWithArgsEnv(output, input, nil, nil)
}

func runWithArgsEnv(
	output io.Writer,
	input io.Reader,
	args []string,
	lookupEnv func(string) (string, bool),
) error {
	options, err := parseRunOptions(args, lookupEnv)
	if err != nil {
		return err
	}

	detect := func() runfx.TTYInfo {
		return runfx.DetectTTYForIO(output, input)
	}

	if options.version {
		_, err := io.WriteString(output, versionString()+"\n")
		return err
	}

	model, err := newApp(output, detect)
	if err != nil {
		return err
	}
	if err := applyRunOptions(model, options, lookupEnv); err != nil {
		return err
	}
	if options.run {
		model.startFlow()
	}

	if options.json || options.quiet || !detect().IsTTY {
		if options.run {
			if err := waitForFlowResult(model, 30*time.Second); err != nil {
				model.Tick(time.Now())
				if writeErr := writeRunOutput(output, model, options); writeErr != nil {
					return writeErr
				}
				return err
			}
		}
		model.Tick(time.Now())
		return writeRunOutput(output, model, options)
	}

	loop, err := runfx.TryStart(runfx.Config{
		Output:       output,
		Input:        input,
		TickInterval: 120 * time.Millisecond,
	})
	if err != nil {
		return err
	}

	if _, err := loop.Mount(model); err != nil {
		return err
	}

	return loop.Run(context.Background())
}

func parseRunOptions(
	args []string,
	lookupEnv func(string) (string, bool),
) (runOptions, error) {
	options := runOptions{}

	if lookupEnv != nil {
		if value, ok := lookupEnv("TFX_FLOW"); ok && strings.TrimSpace(value) != "" {
			options.flow = strings.TrimSpace(value)
		} else if value, ok := lookupEnv("TFX_PROFILE"); ok && strings.TrimSpace(value) != "" {
			options.flow = strings.TrimSpace(value)
		}

		if value, ok := lookupEnv("TFX_LANE"); ok && strings.TrimSpace(value) != "" {
			options.lane = strings.TrimSpace(value)
		}

		if value, ok := lookupEnv("TFX_RUN"); ok && strings.TrimSpace(value) != "" {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid TFX_RUN value %q", value)
			}
			options.run = parsed
		}
		if value, ok := lookupEnv("TFX_JSON"); ok && strings.TrimSpace(value) != "" {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid TFX_JSON value %q", value)
			}
			options.json = parsed
		}
		if value, ok := lookupEnv("TFX_QUIET"); ok && strings.TrimSpace(value) != "" {
			parsed, err := strconv.ParseBool(strings.TrimSpace(value))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid TFX_QUIET value %q", value)
			}
			options.quiet = parsed
		}
	}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "":
			continue
		case strings.HasPrefix(arg, "-test."):
			continue
		case arg == "--":
			i = len(args)
			continue
		case arg == "--flow" || arg == "--profile" || arg == "--lane":
			rest := args[i+1:]
			if len(rest) == 0 {
				return runOptions{}, fmt.Errorf("%s requires a value", arg)
			}
			nextValue := strings.TrimSpace(rest[0])
			i++
			switch arg {
			case "--flow", "--profile":
				options.flow = nextValue
			case "--lane":
				options.lane = nextValue
			}
		case arg == "--run":
			options.run = true
		case arg == "--version":
			options.version = true
		case arg == "--json":
			options.json = true
		case arg == "--quiet":
			options.quiet = true
		case strings.HasPrefix(arg, "--flow="):
			options.flow = strings.TrimSpace(strings.TrimPrefix(arg, "--flow="))
		case strings.HasPrefix(arg, "--profile="):
			options.flow = strings.TrimSpace(strings.TrimPrefix(arg, "--profile="))
		case strings.HasPrefix(arg, "--lane="):
			options.lane = strings.TrimSpace(strings.TrimPrefix(arg, "--lane="))
		case strings.HasPrefix(arg, "--run="):
			parsed, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(arg, "--run=")))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid --run value")
			}
			options.run = parsed
		case strings.HasPrefix(arg, "--version="):
			parsed, err := strconv.ParseBool(strings.TrimSpace(
				strings.TrimPrefix(arg, "--version="),
			))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid --version value")
			}
			options.version = parsed
		case strings.HasPrefix(arg, "--json="):
			parsed, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(arg, "--json=")))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid --json value")
			}
			options.json = parsed
		case strings.HasPrefix(arg, "--quiet="):
			parsed, err := strconv.ParseBool(strings.TrimSpace(strings.TrimPrefix(arg, "--quiet=")))
			if err != nil {
				return runOptions{}, fmt.Errorf("invalid --quiet value")
			}
			options.quiet = parsed
		case strings.HasPrefix(arg, "-"):
			return runOptions{}, fmt.Errorf("unknown flag: %s", arg)
		}
	}

	return options, nil
}

func applyRunOptions(
	model *app,
	options runOptions,
	lookupEnv func(string) (string, bool),
) error {
	if model == nil {
		return nil
	}
	if options.flow != "" {
		if err := model.selectPlanOverride(options.flow); err != nil {
			return err
		}
	}
	if options.lane != "" {
		if err := model.selectLaneOverride(options.lane); err != nil {
			return err
		}
	}
	if options.run {
		return model.prepareAutoRun(lookupEnv)
	}
	return nil
}

func writeRunOutput(output io.Writer, model *app, options runOptions) error {
	if model == nil {
		return nil
	}
	switch {
	case options.json:
		_, err := output.Write(model.jsonSummary())
		return err
	case options.quiet:
		_, err := output.Write(model.quietSummary())
		return err
	default:
		_, err := output.Write(model.RenderSnapshot())
		return err
	}
}

func waitForFlowResult(model *app, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		model.mu.Lock()
		running := model.flowRunning
		done := model.flowDone
		flowError := model.flowError
		model.mu.Unlock()

		switch {
		case running:
			time.Sleep(10 * time.Millisecond)
		case flowError != "":
			return fmt.Errorf("%s", flowError)
		case done:
			return nil
		default:
			time.Sleep(10 * time.Millisecond)
		}
	}

	return fmt.Errorf("timed out waiting for configured flow to finish")
}
